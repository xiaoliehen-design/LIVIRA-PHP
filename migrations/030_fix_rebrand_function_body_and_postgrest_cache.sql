-- ============================================================================
-- LIVIRA HOTFIX 030
-- Perbaikan body function setelah rename SENTRA -> LIVIRA dan reload PostgREST
--
-- Jalankan pada database operasional setelah migration 029.
-- Aman dijalankan berulang kali dan tidak mengubah data inventory.
-- ============================================================================

begin;

-- ALTER FUNCTION ... RENAME mengganti nama object PostgreSQL, tetapi referensi
-- function lain yang ditulis di dalam body SQL/PLpgSQL berbentuk teks tidak
-- selalu ikut ditulis ulang. Akibatnya livira_dashboard_summary() dapat masih
-- memanggil public.sentra_inventory_dashboard_metrics(), sehingga dashboard
-- menghasilkan HTTP 500. Trigger pencarian inventory juga dapat mengalami hal
-- yang sama saat insert/update.
do $$
declare
  object_row record;
  function_definition text;
  repaired_count integer := 0;
begin
  for object_row in
    select p.oid, n.nspname, p.proname
    from pg_proc p
    join pg_namespace n on n.oid = p.pronamespace
    where n.nspname = 'public'
      and p.prokind = 'f'
      and position('livira' in p.proname) > 0
      and position('public.sentra_' in pg_get_functiondef(p.oid)) > 0
    order by p.proname, p.oid
  loop
    function_definition := pg_get_functiondef(object_row.oid);
    function_definition := replace(
      function_definition,
      'public.sentra_',
      'public.livira_'
    );
    execute function_definition;
    repaired_count := repaired_count + 1;
  end loop;

  raise notice 'LIVIRA: % function body diperbaiki.', repaired_count;
end
$$;

-- Pastikan function utama yang dipakai halaman dashboard tersedia.
do $$
begin
  if to_regprocedure('public.livira_dashboard_summary()') is null then
    raise exception 'Function public.livira_dashboard_summary() tidak ditemukan. Jalankan migration 029 fixed terlebih dahulu.';
  end if;

  if to_regprocedure('public.livira_inventory_dashboard_metrics(text)') is null then
    raise exception 'Function public.livira_inventory_dashboard_metrics(text) tidak ditemukan. Pastikan migration dashboard 019/021/027 telah diterapkan sebelum migration 029.';
  end if;
end
$$;

-- CREATE OR REPLACE mempertahankan privilege, tetapi grant berikut memastikan
-- service_role tetap dapat memanggil seluruh RPC LIVIRA yang sudah tersedia.
do $$
declare
  object_row record;
begin
  for object_row in
    select p.oid
    from pg_proc p
    join pg_namespace n on n.oid = p.pronamespace
    where n.nspname = 'public'
      and p.prokind = 'f'
      and p.proname in (
        'livira_dashboard_summary',
        'livira_notification_summary',
        'livira_performance_source',
        'livira_inventory_summary',
        'livira_process_dashboard',
        'livira_create_inventories',
        'livira_create_disposition',
        'livira_apply_disposition_event',
        'livira_apply_inventory_event',
        'livira_correct_inventory_data',
        'livira_relocate_inventory_load'
      )
  loop
    execute format('grant execute on function %s to service_role', object_row.oid::regprocedure);
  end loop;
end
$$;

commit;

-- Paksa PostgREST/Supabase membaca ulang nama dan body RPC terbaru.
notify pgrst, 'reload schema';

-- Hasil harus menunjukkan dua kolom bernilai true.
select
  to_regprocedure('public.livira_dashboard_summary()') is not null
    as livira_dashboard_summary_ready,
  to_regprocedure('public.livira_inventory_dashboard_metrics(text)') is not null
    as livira_dashboard_metrics_ready;
