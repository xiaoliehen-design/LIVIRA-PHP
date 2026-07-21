-- ============================================================================
-- LIVIRA — FILTER KPI DASHBOARD DAN PINDAH/BONGKAR KONTAINER
-- Migration 028
--
-- Dashboard scope dihitung oleh aplikasi. Migration ini menambahkan operasi
-- database atomik untuk memindahkan satu uraian barang ke satu atau beberapa
-- kontainer tujuan dan/atau membongkarnya menjadi LCL.
-- Aman dijalankan ulang setelah seluruh migration sebelumnya.
-- ============================================================================

begin;

create or replace function public.livira_relocate_inventory_load(
  p_inventory_id uuid,
  p_expected_updated_at timestamptz,
  p_allocations jsonb,
  p_event jsonb default '{}'::jsonb
)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_source public.inventory_items%rowtype;
  v_clone public.inventory_items%rowtype;
  v_allocation jsonb;
  v_normalized jsonb := '[]'::jsonb;
  v_result_ids uuid[] := array[]::uuid[];
  v_affected_units text[] := array[]::text[];
  v_seen_containers text[] := array[]::text[];
  v_now timestamptz := now();
  v_index integer := 0;
  v_count integer := 0;
  v_load_type text;
  v_container_compact text;
  v_container_no text;
  v_container_size text;
  v_quantity numeric(18,2);
  v_volume numeric(14,2);
  v_total_quantity numeric(18,2) := 0;
  v_allocated_value bigint := 0;
  v_goods_value bigint;
  v_remaining_value bigint;
  v_physical_unit_id text;
  v_old_unit_id text;
  v_new_id uuid;
  v_primary_id uuid;
  v_unit_id text;
  v_actor text := trim(coalesce(p_event->>'actor', ''));
  v_document_no text := trim(coalesce(p_event->>'document_no', ''));
  v_document_date timestamptz;
  v_result jsonb;
begin
  select * into v_source
  from public.inventory_items
  where id = p_inventory_id
  for update;

  if not found then
    raise exception 'not found: inventory' using errcode = 'P0002';
  end if;
  if not v_source.is_active then
    raise exception 'inventory is inactive' using errcode = 'P0001';
  end if;
  if v_source.current_disposition is not null
     or v_source.status_code in ('laku', 'alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima')
     or coalesce(v_source.quantity, 0) <= 0 then
    raise exception 'invalid transition: inventory cannot be relocated' using errcode = 'P0001';
  end if;
  if p_expected_updated_at is not null
     and v_source.updated_at is distinct from p_expected_updated_at then
    raise exception 'record changed by another user' using errcode = '40001';
  end if;
  if v_actor = '' or v_document_no = '' or trim(coalesce(p_event->>'document_date', '')) = '' then
    raise exception 'invalid transition: document and actor are required' using errcode = 'P0001';
  end if;

  begin
    v_document_date := (p_event->>'document_date')::timestamptz;
  exception when others then
    raise exception 'invalid transition: invalid document date' using errcode = 'P0001';
  end;

  if jsonb_typeof(p_allocations) <> 'array' then
    raise exception 'invalid transition: allocations must be an array' using errcode = 'P0001';
  end if;
  v_count := jsonb_array_length(p_allocations);
  if v_count < 1 or v_count > 20 then
    raise exception 'invalid transition: allocation count must be between 1 and 20' using errcode = 'P0001';
  end if;

  for v_allocation in select value from jsonb_array_elements(p_allocations)
  loop
    v_load_type := upper(trim(coalesce(v_allocation->>'load_type', '')));

    if trim(coalesce(v_allocation->>'quantity', '')) !~ '^[0-9]+([.][0-9]+)?$' then
      raise exception 'invalid transition: invalid allocation quantity' using errcode = 'P0001';
    end if;
    v_quantity := round((v_allocation->>'quantity')::numeric, 2);
    if v_quantity <= 0 then
      raise exception 'invalid transition: allocation quantity must be positive' using errcode = 'P0001';
    end if;

    if v_load_type = 'FCL' then
      v_container_compact := upper(regexp_replace(coalesce(v_allocation->>'container_no', ''), '[^A-Za-z0-9]', '', 'g'));
      v_container_size := upper(trim(coalesce(v_allocation->>'container_size', '')));
      if v_container_compact !~ '^[A-Z]{4}[0-9]{7}$'
         or v_container_size not in ('20', '40', '40HC', '45HC') then
        raise exception 'invalid transition: invalid destination container' using errcode = 'P0001';
      end if;
      if v_container_compact = any(v_seen_containers) then
        raise exception 'invalid transition: duplicate destination container' using errcode = 'P0001';
      end if;
      v_seen_containers := array_append(v_seen_containers, v_container_compact);
      v_container_no := substr(v_container_compact, 1, 4) || ' ' || substr(v_container_compact, 5, 6) || '-' || substr(v_container_compact, 11, 1);
      v_volume := 0;
    elsif v_load_type = 'LCL' then
      if trim(coalesce(v_allocation->>'estimated_volume_m3', '')) !~ '^[0-9]+([.][0-9]+)?$' then
        raise exception 'invalid transition: invalid LCL volume' using errcode = 'P0001';
      end if;
      v_volume := round((v_allocation->>'estimated_volume_m3')::numeric, 2);
      if v_volume <= 0 then
        raise exception 'invalid transition: LCL volume must be positive' using errcode = 'P0001';
      end if;
      v_container_compact := '';
      v_container_no := '';
      v_container_size := '';
    else
      raise exception 'invalid transition: load type must be FCL or LCL' using errcode = 'P0001';
    end if;

    v_total_quantity := v_total_quantity + v_quantity;
    v_normalized := v_normalized || jsonb_build_array(jsonb_build_object(
      'load_type', v_load_type,
      'container_no', v_container_no,
      'container_compact', v_container_compact,
      'container_size', v_container_size,
      'estimated_volume_m3', v_volume,
      'quantity', v_quantity
    ));
  end loop;

  if abs(v_total_quantity - v_source.quantity) > 0.005 then
    raise exception 'invalid transition: allocation quantity must equal source quantity' using errcode = 'P0001';
  end if;

  if v_count = 1 then
    v_allocation := v_normalized->0;
    if upper(trim(coalesce(v_source.load_type, ''))) = v_allocation->>'load_type' then
      if v_allocation->>'load_type' = 'FCL'
         and upper(regexp_replace(coalesce(v_source.container_no, ''), '[^A-Za-z0-9]', '', 'g')) = v_allocation->>'container_compact'
         and upper(trim(coalesce(v_source.container_size, ''))) = v_allocation->>'container_size' then
        raise exception 'invalid transition: destination is unchanged' using errcode = 'P0001';
      elsif v_allocation->>'load_type' = 'LCL'
            and abs(coalesce(v_source.estimated_volume_m3, 0) - (v_allocation->>'estimated_volume_m3')::numeric) <= 0.005 then
        raise exception 'invalid transition: destination is unchanged' using errcode = 'P0001';
      end if;
    end if;
  end if;

  v_old_unit_id := coalesce(nullif(trim(v_source.physical_unit_id), ''), v_source.id::text);
  v_affected_units := array_append(v_affected_units, v_old_unit_id);

  for v_allocation in select value from jsonb_array_elements(v_normalized)
  loop
    v_index := v_index + 1;
    v_load_type := v_allocation->>'load_type';
    v_container_no := v_allocation->>'container_no';
    v_container_compact := v_allocation->>'container_compact';
    v_container_size := v_allocation->>'container_size';
    v_quantity := (v_allocation->>'quantity')::numeric;
    v_volume := (v_allocation->>'estimated_volume_m3')::numeric;

    if v_index = v_count then
      v_goods_value := greatest(v_source.goods_value - v_allocated_value, 0);
    else
      v_remaining_value := greatest(v_source.goods_value - v_allocated_value, 0);
      v_goods_value := least(
        greatest(round(v_source.goods_value::numeric * v_quantity / v_source.quantity)::bigint, 0),
        v_remaining_value
      );
      v_allocated_value := v_allocated_value + v_goods_value;
    end if;

    if v_load_type = 'LCL' then
      v_physical_unit_id := 'LCL:' || gen_random_uuid()::text;
    else
      if upper(regexp_replace(coalesce(v_source.container_no, ''), '[^A-Za-z0-9]', '', 'g')) = v_container_compact
         and trim(coalesce(v_source.physical_unit_id, '')) <> '' then
        v_physical_unit_id := trim(v_source.physical_unit_id);
      else
        select coalesce(nullif(trim(i.physical_unit_id), ''), 'FCL:' || v_container_compact)
        into v_physical_unit_id
        from public.inventory_items i
        where i.is_active = true
          and upper(coalesce(i.load_type, '')) = 'FCL'
          and i.facility_id is not distinct from v_source.facility_id
          and i.at_tpp = v_source.at_tpp
          and upper(regexp_replace(coalesce(i.container_no, ''), '[^A-Za-z0-9]', '', 'g')) = v_container_compact
        order by i.occupancy_primary desc, i.created_at, i.id
        limit 1;
        v_physical_unit_id := coalesce(v_physical_unit_id, 'FCL:' || v_container_compact);
      end if;
    end if;

    if not (v_physical_unit_id = any(v_affected_units)) then
      v_affected_units := array_append(v_affected_units, v_physical_unit_id);
    end if;

    if v_index = 1 then
      update public.inventory_items
      set load_type = v_load_type,
          container_no = v_container_no,
          container_size = v_container_size,
          estimated_volume_m3 = v_volume,
          physical_unit_id = v_physical_unit_id,
          occupancy_primary = false,
          quantity = v_quantity,
          goods_value = v_goods_value,
          status_code = 'pindah_bongkar_kontainer',
          status_label = 'Pindah/Bongkar Kontainer',
          updated_at = v_now
      where id = v_source.id;
      v_new_id := v_source.id;
    else
      v_new_id := gen_random_uuid();
      v_clone := v_source;
      v_clone.id := v_new_id;
      v_clone.reference_no := v_source.reference_no || '/MOVE-' || lpad(v_index::text, 2, '0') || '-' || substr(replace(v_new_id::text, '-', ''), 1, 8);
      v_clone.load_type := v_load_type;
      v_clone.container_no := v_container_no;
      v_clone.container_size := v_container_size;
      v_clone.estimated_volume_m3 := v_volume;
      v_clone.physical_unit_id := v_physical_unit_id;
      v_clone.occupancy_primary := false;
      v_clone.quantity := v_quantity;
      v_clone.goods_value := v_goods_value;
      v_clone.status_code := 'pindah_bongkar_kontainer';
      v_clone.status_label := 'Pindah/Bongkar Kontainer';
      v_clone.current_disposition := null;
      v_clone.is_active := true;
      v_clone.created_by := v_actor;
      v_clone.created_at := v_now;
      v_clone.updated_at := v_now;
      v_clone.search_text := '';

      insert into public.inventory_items
      select (v_clone).*;

      insert into public.events (
        inventory_id, disposition_id, disposition_type,
        code, label, document_no, document_date,
        notes, actor, created_at, document_id
      )
      select
        v_new_id, null, null,
        e.code, e.label, e.document_no, e.document_date,
        e.notes, e.actor, e.created_at, e.document_id
      from public.events e
      where e.inventory_id = v_source.id
      order by e.created_at, e.id;
    end if;

    v_result_ids := array_append(v_result_ids, v_new_id);
  end loop;

  insert into public.events (
    inventory_id, code, label, document_no, document_date,
    notes, actor, created_at, document_id
  )
  select
    x.inventory_id,
    'pindah_bongkar_kontainer',
    'Pindah/Bongkar Kontainer',
    v_document_no,
    v_document_date,
    trim(coalesce(p_event->>'notes', '')),
    v_actor,
    v_now,
    nullif(trim(coalesce(p_event->>'document_id', '')), '')::uuid
  from unnest(v_result_ids) as x(inventory_id);

  foreach v_unit_id in array v_affected_units
  loop
    update public.inventory_items i
    set occupancy_primary = false
    where i.is_active = true
      and coalesce(nullif(trim(i.physical_unit_id), ''), i.id::text) = v_unit_id;

    select i.id into v_primary_id
    from public.inventory_items i
    where i.is_active = true
      and coalesce(nullif(trim(i.physical_unit_id), ''), i.id::text) = v_unit_id
    order by i.created_at, i.id
    limit 1;

    if v_primary_id is not null then
      update public.inventory_items
      set occupancy_primary = true
      where id = v_primary_id;
    end if;
    v_primary_id := null;
  end loop;

  select coalesce(
    jsonb_agg(to_jsonb(i) - 'search_text' order by x.ordinality),
    '[]'::jsonb
  ) into v_result
  from unnest(v_result_ids) with ordinality as x(inventory_id, ordinality)
  join public.inventory_items i on i.id = x.inventory_id;

  return v_result;
end;
$$;

revoke all on function public.livira_relocate_inventory_load(uuid, timestamptz, jsonb, jsonb)
  from public, anon, authenticated;
grant execute on function public.livira_relocate_inventory_load(uuid, timestamptz, jsonb, jsonb)
  to service_role;

comment on function public.livira_relocate_inventory_load(uuid, timestamptz, jsonb, jsonb) is
  'Memindahkan satu uraian inventory secara atomik ke beberapa kontainer dan/atau LCL sambil menjaga total kuantitas, total nilai, timeline, dan occupancy YOR/SOR.';

commit;
