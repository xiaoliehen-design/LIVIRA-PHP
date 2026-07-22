-- ============================================================================
-- LIVIRA — SETUP DATABASE BARU KOSONG (REVISI APLIKASI 033 / SKEMA 001 s.d. 032)
-- ============================================================================
-- Revisi aplikasi 033 tidak membutuhkan migration database baru; fitur lupa
-- password, CAPTCHA, dan hapus user menggunakan skema sampai migration 032.
--
-- Tujuan:
--   1. Menyiapkan seluruh skema, role, parameter, workflow, keamanan,
--      private Storage, audit, pencarian, dan RPC performa LIVIRA.
--   2. TIDAK memasukkan inventory/barang, proses, timeline, atau data contoh.
--
-- Jalankan file ini SATU KALI pada Supabase SQL Editor untuk project baru.
-- File migrations/002_seed.sql sengaja TIDAK disertakan karena berisi data
-- barang contoh/dummy.
--
-- Master TPP yang dibuat pada bagian akhir:
--   - TPP Transporindo
--   - TPP Multi Sejahtera
--   - TPP KBN Marunda
--   - TPP Graha Segara
-- Kapasitas awal YOR/SOR dan pemakaian diatur 0 agar dapat diisi melalui menu
-- Dashboard sesuai kapasitas riil.
-- ============================================================================


-- ============================================================================
-- BEGIN MIGRATION: 001_schema.sql
-- ============================================================================
-- ================================================================
-- LIVIRA — SKEMA SUPABASE / POSTGRESQL
-- Jalankan seluruh file ini melalui Supabase SQL Editor.
-- ================================================================

begin;

create extension if not exists pgcrypto;

create table if not exists public.facilities (
  id text primary key,
  name text not null unique,
  active boolean not null default true,
  sort_order smallint not null default 999,
  yard_capacity numeric(14,2) not null default 0 check (yard_capacity >= 0),
  yard_used numeric(14,2) not null default 0 check (yard_used >= 0),
  shed_capacity numeric(14,2) not null default 0 check (shed_capacity >= 0),
  shed_used numeric(14,2) not null default 0 check (shed_used >= 0),
  created_at timestamptz not null default now()
);

alter table public.facilities
  add column if not exists sort_order smallint not null default 999,
  add column if not exists yard_capacity numeric(14,2) not null default 0,
  add column if not exists yard_used numeric(14,2) not null default 0,
  add column if not exists shed_capacity numeric(14,2) not null default 0,
  add column if not exists shed_used numeric(14,2) not null default 0;

create table if not exists public.inventory_items (
  id uuid primary key default gen_random_uuid(),
  reference_no text not null unique,
  item_type text not null check (item_type in ('BTD','BDN','BMMN')),
  origin_type text not null check (origin_type in ('BTD','BDN')),
  manifest_no text not null default '',
  manifest_date timestamptz,
  manifest_position text not null default '',
  determination_no text not null,
  determination_date timestamptz not null,
  category text not null default '',
  description text not null,
  item_kind text not null default 'Barang Umum' constraint inventory_item_kind_check check (item_kind in ('Barang Umum','Barang Berbahaya (B3)','Hewan atau Tumbuhan Hidup','Barang Peka Waktu','Barang Berharga')),
  quantity numeric(18,2) not null default 0 check (quantity >= 0),
  unit text not null default '',
  goods_value bigint not null default 0 check (goods_value >= 0),
  location text not null default '',
  location_status text not null default 'Masih di TPS',
  at_tpp boolean not null default false,
  owner_name text not null default '',
  owner_address text not null default '',
  origin_warehouse text not null default '',
  facility_id text references public.facilities(id) on update cascade,
  facility_name text not null default '',
  load_type text not null default 'FCL' check (load_type in ('FCL','LCL')),
  container_no text not null default '',
  container_size text not null default '' check (container_size in ('', '20', '40', '45')),
  estimated_volume_m3 numeric(14,2) not null default 0 check (estimated_volume_m3 >= 0),
  research_request_no text not null default '',
  research_request_date timestamptz,
  hs_code text not null default '',
  is_restricted boolean not null default false,
  restriction_rule text not null default '',
  origin_document_type text not null default '',
  origin_document_no text not null default '',
  origin_document_date timestamptz,
  allocation_purpose text not null default '',
  allocation_proposal_type text not null default '',
  allocation_proposal_no text not null default '',
  allocation_proposal_date timestamptz,
  allocation_approval_type text not null default '',
  allocation_approval_no text not null default '',
  allocation_approval_date timestamptz,
  exit_document_no text not null default '',
  exit_document_date timestamptz,
  exit_type text not null default '',
  exit_notes text not null default '',
  status_code text not null default 'ditetapkan',
  status_label text not null default 'Ditetapkan',
  current_disposition text check (current_disposition is null or current_disposition in ('lelang','musnah','hibah')),
  is_active boolean not null default true,
  created_by text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.dispositions (
  id uuid primary key default gen_random_uuid(),
  inventory_id uuid not null references public.inventory_items(id) on delete restrict,
  disposition_type text not null check (disposition_type in ('lelang','musnah','hibah')),
  round integer not null default 1 check (round between 1 and 99),
  status_code text not null,
  status_label text not null,
  proposal_type text not null default '',
  recipient_code text not null default '',
  recipient_name text not null default '',
  sale_value bigint not null default 0 check (sale_value >= 0),
  htl_value bigint not null default 0 check (htl_value >= 0),
  auction_cost bigint not null default 0 check (auction_cost >= 0),
  execution_start_date timestamptz,
  execution_end_date timestamptz,
  auction_outcome text not null default '',
  allocation_target text not null default '',
  destruction_cost bigint not null default 0 check (destruction_cost >= 0),
  transfer_type text not null default '',
  is_active boolean not null default true,
  created_by text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.events (
  id uuid primary key default gen_random_uuid(),
  inventory_id uuid not null references public.inventory_items(id) on delete cascade,
  disposition_id uuid references public.dispositions(id) on delete cascade,
  disposition_type text check (disposition_type is null or disposition_type in ('lelang','musnah','hibah')),
  code text not null,
  label text not null,
  document_no text not null default '',
  document_date timestamptz,
  notes text not null default '',
  actor text not null,
  created_at timestamptz not null default now()
);

create index if not exists inventory_active_idx
  on public.inventory_items (is_active, item_type, facility_id, updated_at desc);

create index if not exists inventory_container_idx
  on public.inventory_items (container_no);

create index if not exists inventory_reference_idx
  on public.inventory_items (reference_no);

create index if not exists inventory_determination_idx
  on public.inventory_items (determination_no);

create index if not exists inventory_status_idx
  on public.inventory_items (status_code, updated_at desc);

create index if not exists inventory_allocation_purpose_idx
  on public.inventory_items (allocation_purpose, is_active, determination_date desc);

create index if not exists disposition_type_status_idx
  on public.dispositions (disposition_type, is_active, updated_at desc);

create unique index if not exists one_active_disposition_per_inventory_idx
  on public.dispositions (inventory_id)
  where is_active = true;

create index if not exists events_inventory_time_idx
  on public.events (inventory_id, created_at asc);

create or replace function public.set_updated_at()
returns trigger
language plpgsql
set search_path = public
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

drop trigger if exists inventory_set_updated_at on public.inventory_items;
create trigger inventory_set_updated_at
before update on public.inventory_items
for each row execute function public.set_updated_at();

drop trigger if exists dispositions_set_updated_at on public.dispositions;
create trigger dispositions_set_updated_at
before update on public.dispositions
for each row execute function public.set_updated_at();

-- Seluruh operasi data aplikasi dilakukan oleh backend menggunakan service role.
-- Anon/authenticated tidak diberi akses langsung ke tabel agar aturan workflow
-- tidak dapat dilewati dari browser.
alter table public.facilities enable row level security;
alter table public.inventory_items enable row level security;
alter table public.dispositions enable row level security;
alter table public.events enable row level security;

revoke all on table public.facilities from anon, authenticated;
revoke all on table public.inventory_items from anon, authenticated;
revoke all on table public.dispositions from anon, authenticated;
revoke all on table public.events from anon, authenticated;

commit;

-- END MIGRATION: 001_schema.sql

-- ============================================================================
-- BEGIN MIGRATION: 003_livira_upgrade.sql
-- ============================================================================
-- ================================================================
-- UPGRADE SIPANDAI TPP LAMA KE LIVIRA
-- Jalankan sekali di Supabase SQL Editor untuk database yang sudah ada.
-- Aman dijalankan ulang.
-- ================================================================

begin;

alter table public.facilities
  add column if not exists yard_capacity integer not null default 0,
  add column if not exists yard_used integer not null default 0,
  add column if not exists shed_capacity integer not null default 0,
  add column if not exists shed_used integer not null default 0;

update public.facilities set yard_capacity = 1250, yard_used = 812, shed_capacity = 4800, shed_used = 3010 where id = 'tpp-transporindo';
update public.facilities set yard_capacity = 980, yard_used = 574, shed_capacity = 3600, shed_used = 2165 where id = 'tpp-multi-sejahtera';
update public.facilities set yard_capacity = 1450, yard_used = 963, shed_capacity = 5200, shed_used = 3484 where id = 'tpp-kbn-marunda';
update public.facilities set yard_capacity = 1100, yard_used = 621, shed_capacity = 4100, shed_used = 2398 where id = 'tpp-graha-segara';

alter table public.inventory_items
  add column if not exists item_kind text not null default 'Barang Umum',
  add column if not exists goods_value bigint not null default 0,
  add column if not exists location_status text not null default 'Masih di TPS',
  add column if not exists at_tpp boolean not null default false,
  add column if not exists research_request_no text not null default '',
  add column if not exists research_request_date timestamptz,
  add column if not exists hs_code text not null default '',
  add column if not exists is_restricted boolean not null default false,
  add column if not exists allocation_purpose text not null default '',
  add column if not exists exit_document_no text not null default '',
  add column if not exists exit_document_date timestamptz,
  add column if not exists exit_type text not null default '',
  add column if not exists exit_notes text not null default '';

alter table public.inventory_items alter column facility_id drop not null;
alter table public.inventory_items alter column facility_name set default '';

update public.inventory_items
set at_tpp = facility_id is not null,
    location_status = case when facility_id is null then 'Masih di TPS' else 'Berada di ' || facility_name end
where location_status = '' or location_status = 'Masih di TPS';

update public.inventory_items
set status_code = 'penelitian_pfpd', status_label = 'Penelitian PFPD'
where status_code = 'penelitian_hs_lartas';

update public.inventory_items
set status_code = 'request_penelitian_pfpd', status_label = 'Request Penelitian PFPD'
where status_code = 'siap_peruntukan';

create index if not exists inventory_determination_idx
  on public.inventory_items (determination_no);

commit;

-- END MIGRATION: 003_livira_upgrade.sql

-- ============================================================================
-- BEGIN MIGRATION: 004_workflow_revision.sql
-- ============================================================================
-- ================================================================
-- LIVIRA — REVISI MENU ACTION DAN DOKUMEN PROSES
-- Jalankan setelah 003_livira_upgrade.sql pada database lama.
-- Aman dijalankan ulang.
-- ================================================================

begin;

alter table public.inventory_items
  add column if not exists origin_document_type text not null default '',
  add column if not exists origin_document_no text not null default '',
  add column if not exists origin_document_date timestamptz,
  add column if not exists allocation_proposal_type text not null default '',
  add column if not exists allocation_proposal_no text not null default '',
  add column if not exists allocation_proposal_date timestamptz,
  add column if not exists allocation_approval_type text not null default '',
  add column if not exists allocation_approval_no text not null default '',
  add column if not exists allocation_approval_date timestamptz;

alter table public.dispositions
  add column if not exists htl_value bigint not null default 0,
  add column if not exists auction_cost bigint not null default 0,
  add column if not exists execution_start_date timestamptz,
  add column if not exists execution_end_date timestamptz,
  add column if not exists auction_outcome text not null default '',
  add column if not exists allocation_target text not null default '',
  add column if not exists destruction_cost bigint not null default 0,
  add column if not exists transfer_type text not null default '';

alter table public.dispositions drop constraint if exists dispositions_round_check;
alter table public.dispositions
  add constraint dispositions_round_check check (round between 1 and 99);

update public.inventory_items
set origin_document_type = case when origin_type = 'BDN' then 'KEP BDN' else 'BCF 1.5' end,
    origin_document_no = coalesce(
      nullif((select event.document_no from public.events event where event.inventory_id = inventory_items.id and event.code in ('ditetapkan', 'masih_di_tps') order by event.created_at asc limit 1), ''),
      determination_no
    ),
    origin_document_date = coalesce(
      (select event.document_date from public.events event where event.inventory_id = inventory_items.id and event.code in ('ditetapkan', 'masih_di_tps') order by event.created_at asc limit 1),
      determination_date
    )
where item_type = 'BMMN'
  and origin_document_no = '';

update public.inventory_items
set category = ''
where item_type = 'BTD';

update public.inventory_items
set origin_warehouse = case origin_warehouse
      when 'JICT' then 'PT Agung Raya'
      when 'KOJA' then 'PT Indonesian Air & Marine Supply (Utara)'
      when 'NPCT1' then 'PT Pelabuhan Indonesia II (Persero) (NPCT1)'
      when 'MAL' then 'PT Multi Terminal Indonesia (CDC Banda)'
      else origin_warehouse
    end;

update public.inventory_items
set location = origin_warehouse
where at_tpp = false;

update public.inventory_items
set unit = 'Piece'
where unit in ('Package', 'Unit');

update public.dispositions
set status_code = case status_code
      when 'usulan_lelang' then 'kep_lelang'
      when 'persetujuan_lelang' then 'kep_lelang'
      when 'penetapan_htl' then 'kep_htl'
      when 'lelang_tidak_laku' then 'tidak_laku'
      when 'alokasi_hasil' then 'alokasi_hasil_lelang'
      else status_code
    end,
    status_label = case status_code
      when 'usulan_lelang' then 'Mulai lelang'
      when 'persetujuan_lelang' then 'Mulai lelang'
      when 'penetapan_htl' then 'HTL ditetapkan'
      when 'lelang_tidak_laku' then 'Tidak laku'
      when 'alokasi_hasil' then 'Alokasi Hasil Lelang'
      else status_label
    end
where disposition_type = 'lelang';

update public.dispositions
set status_label = 'Jadwal lelang ditetapkan'
where disposition_type = 'lelang'
  and status_code = 'jadwal_lelang';

update public.dispositions
set status_code = case when sale_value > 0 then 'laku' else 'tidak_laku' end,
    status_label = case when sale_value > 0 then 'Laku' else 'Tidak laku' end,
    auction_outcome = case when sale_value > 0 then 'laku' else 'tidak_laku' end
where disposition_type = 'lelang'
  and status_code = 'risalah_lelang';

update public.dispositions
set is_active = false
where status_code in ('alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima');

update public.inventory_items item
set current_disposition = null
where exists (
    select 1 from public.dispositions process
    where process.inventory_id = item.id
      and process.status_code in ('alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima')
  )
  and not exists (
    select 1 from public.dispositions process
    where process.inventory_id = item.id
      and process.is_active = true
  );

update public.dispositions
set status_code = 'kep_musnah',
    status_label = 'KEP Musnah diterbitkan'
where disposition_type = 'musnah'
  and status_code in ('usulan_musnah', 'persetujuan_musnah');

-- Workflow Hibah/PSP kini hanya BA Serah Terima. Proses lama dibuka kembali
-- ke inventory agar petugas dapat mencatat BA melalui action baru.
update public.dispositions
set is_active = false,
    status_code = 'workflow_lama_hibah',
    status_label = 'Menunggu BA Serah Terima pada workflow baru'
where disposition_type = 'hibah'
  and is_active = true
  and status_code <> 'ba_serah_terima';

update public.dispositions
set is_active = false
where disposition_type = 'hibah'
  and status_code = 'ba_serah_terima';

update public.inventory_items item
set current_disposition = null,
    status_code = 'siap_ba_serah_terima',
    status_label = 'Siap dicatat BA Serah Terima'
where item.current_disposition = 'hibah'
  and not exists (
    select 1 from public.dispositions process
    where process.inventory_id = item.id
      and process.is_active = true
  );

update public.inventory_items item
set status_code = process.status_code,
    status_label = process.status_label
from public.dispositions process
where process.inventory_id = item.id
  and process.is_active = true;

commit;

-- END MIGRATION: 004_workflow_revision.sql

-- ============================================================================
-- BEGIN MIGRATION: 005_reporting_item_kind.sql
-- ============================================================================
-- LIVIRA — revisi jenis barang dan dukungan pelaporan
-- Jalankan sekali setelah 004_workflow_revision.sql pada database yang sudah ada.

begin;

update public.inventory_items
set item_kind = case
  when item_kind in (
    'Barang Umum',
    'Barang Berbahaya (B3)',
    'Hewan atau Tumbuhan Hidup',
    'Barang Peka Waktu',
    'Barang Berharga'
  ) then item_kind
  else 'Barang Umum'
end;

alter table public.inventory_items
  alter column item_kind set default 'Barang Umum';

alter table public.inventory_items
  drop constraint if exists inventory_item_kind_check;

alter table public.inventory_items
  add constraint inventory_item_kind_check check (
    item_kind in (
      'Barang Umum',
      'Barang Berbahaya (B3)',
      'Hewan atau Tumbuhan Hidup',
      'Barang Peka Waktu',
      'Barang Berharga'
    )
  );

create index if not exists inventory_reporting_date_idx
  on public.inventory_items (determination_date, is_active, goods_value desc);

commit;

-- END MIGRATION: 005_reporting_item_kind.sql

-- ============================================================================
-- BEGIN MIGRATION: 006_history_search_dashboard.sql
-- ============================================================================
-- ================================================================
-- LIVIRA — HISTORY, PENCARIAN DETAIL, DAN DASHBOARD PROSES
-- Jalankan setelah 005_reporting_item_kind.sql pada database lama.
-- Aman dijalankan ulang.
-- ================================================================

begin;

alter table public.dispositions
  add column if not exists auction_cost bigint not null default 0;

alter table public.dispositions
  drop constraint if exists dispositions_auction_cost_check;

alter table public.dispositions
  add constraint dispositions_auction_cost_check check (auction_cost >= 0);

-- Normalisasi peruntukan lama agar filter BMMN menggunakan empat pilihan baku.
update public.inventory_items
set allocation_purpose = case
  when lower(trim(allocation_purpose)) = 'lelang' then 'Lelang'
  when lower(trim(allocation_purpose)) in ('musnah', 'pemusnahan') then 'Musnah'
  when lower(trim(allocation_purpose)) = 'hibah' then 'Hibah'
  when lower(trim(allocation_purpose)) = 'psp' then 'PSP'
  else allocation_purpose
end
where trim(allocation_purpose) <> '';

create index if not exists inventory_allocation_purpose_idx
  on public.inventory_items (allocation_purpose, is_active, determination_date desc);

create index if not exists inventory_history_idx
  on public.inventory_items (is_active, updated_at desc, determination_date desc);

create index if not exists disposition_dashboard_idx
  on public.dispositions (disposition_type, created_at desc, is_active);

commit;

-- END MIGRATION: 006_history_search_dashboard.sql

-- ============================================================================
-- BEGIN MIGRATION: 007_access_approval_parameters.sql
-- ============================================================================
-- =====================================================================
-- LIVIRA — OTP, PERSETUJUAN PENDAFTARAN, ROLE, DAN PARAMETER DINAMIS
-- Jalankan setelah 006_history_search_dashboard.sql.
-- Aman dijalankan ulang.
-- =====================================================================

begin;

create table if not exists public.app_roles (
  id uuid primary key default gen_random_uuid(),
  name text not null unique,
  description text not null default '',
  permissions jsonb not null default '[]'::jsonb check (jsonb_typeof(permissions) = 'array'),
  active boolean not null default true,
  system boolean not null default false,
  created_by text not null default 'system',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.app_users (
  id uuid primary key default gen_random_uuid(),
  auth_user_id uuid not null unique references auth.users(id) on delete cascade,
  name text not null default '',
  email text not null,
  email_verified boolean not null default false,
  email_verified_at timestamptz,
  approval_status text not null default 'pending'
    check (approval_status in ('pending','approved','rejected')),
  role_id uuid references public.app_roles(id) on delete restrict,
  rejection_reason text not null default '',
  approved_by text not null default '',
  approved_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create unique index if not exists app_users_email_lower_idx
  on public.app_users (lower(email));
create index if not exists app_users_approval_idx
  on public.app_users (approval_status, email_verified, created_at desc);
create index if not exists app_users_role_idx
  on public.app_users (role_id, approval_status);

create table if not exists public.app_parameters (
  id uuid primary key default gen_random_uuid(),
  group_code text not null
    check (group_code in ('bdn_category','item_kind','exit_type')),
  code text not null,
  label text not null,
  applies_to text not null default '',
  active boolean not null default true,
  system boolean not null default false,
  sort_order integer not null default 999,
  created_by text not null default 'system',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (group_code, code)
);

create index if not exists app_parameters_active_idx
  on public.app_parameters (group_code, active, sort_order, label);

-- Trigger timestamp menggunakan fungsi public.set_updated_at dari migration 001.
drop trigger if exists app_roles_set_updated_at on public.app_roles;
create trigger app_roles_set_updated_at
before update on public.app_roles
for each row execute function public.set_updated_at();

drop trigger if exists app_users_set_updated_at on public.app_users;
create trigger app_users_set_updated_at
before update on public.app_users
for each row execute function public.set_updated_at();

drop trigger if exists app_parameters_set_updated_at on public.app_parameters;
create trigger app_parameters_set_updated_at
before update on public.app_parameters
for each row execute function public.set_updated_at();

-- Profil pendaftaran otomatis dibuat begitu Supabase Auth menerima sign-up.
create or replace function public.handle_livira_new_auth_user()
returns trigger
language plpgsql
security definer
set search_path = public, auth
as $$
begin
  insert into public.app_users (
    auth_user_id,
    name,
    email,
    email_verified,
    email_verified_at,
    approval_status
  ) values (
    new.id,
    coalesce(nullif(trim(new.raw_user_meta_data ->> 'name'), ''), split_part(coalesce(new.email, ''), '@', 1)),
    lower(coalesce(new.email, '')),
    new.email_confirmed_at is not null,
    new.email_confirmed_at,
    'pending'
  )
  on conflict (auth_user_id) do update set
    name = excluded.name,
    email = excluded.email,
    email_verified = excluded.email_verified,
    email_verified_at = excluded.email_verified_at,
    updated_at = now();
  return new;
end;
$$;

drop trigger if exists livira_auth_user_created on auth.users;
create trigger livira_auth_user_created
after insert on auth.users
for each row execute function public.handle_livira_new_auth_user();

create or replace function public.handle_livira_auth_confirmation()
returns trigger
language plpgsql
security definer
set search_path = public, auth
as $$
begin
  if new.email_confirmed_at is not null
     and old.email_confirmed_at is distinct from new.email_confirmed_at then
    update public.app_users
    set email_verified = true,
        email_verified_at = new.email_confirmed_at,
        email = lower(coalesce(new.email, email)),
        updated_at = now()
    where auth_user_id = new.id;
  end if;
  return new;
end;
$$;

drop trigger if exists livira_auth_user_confirmed on auth.users;
create trigger livira_auth_user_confirmed
after update of email_confirmed_at, email on auth.users
for each row execute function public.handle_livira_auth_confirmation();

-- Backfill akun Auth yang sudah ada sebelum migration ini.
insert into public.app_users (
  auth_user_id, name, email, email_verified, email_verified_at, approval_status
)
select
  u.id,
  coalesce(nullif(trim(u.raw_user_meta_data ->> 'name'), ''), split_part(coalesce(u.email, ''), '@', 1)),
  lower(coalesce(u.email, '')),
  u.email_confirmed_at is not null,
  u.email_confirmed_at,
  'pending'
from auth.users u
where coalesce(u.email, '') <> ''
on conflict (auth_user_id) do nothing;

-- Role awal. Administrator utama tetap berasal dari ADMIN_USERNAME/ADMIN_PASSWORD.
insert into public.app_roles (name, description, permissions, active, system, created_by)
values
  ('Petugas Inventory', 'Kelola inventory BTD, BDN, dan BMMN.',
   '["dashboard.view","inventory.view","inventory.manage","inventory.type.btd","inventory.type.bdn","inventory.type.bmmn","reports.view","search.view"]'::jsonb, true, true, 'migration'),
  ('Petugas Lelang', 'Akses khusus proses dan dashboard lelang.',
   '["dashboard.view","auction.view","auction.manage","inventory.type.btd","inventory.type.bdn","inventory.type.bmmn","search.view"]'::jsonb, true, true, 'migration'),
  ('Petugas BMMN', 'Akses inventory dan pelaporan khusus BMMN.',
   '["dashboard.view","inventory.view","inventory.manage","inventory.type.bmmn","reports.view","search.view"]'::jsonb, true, true, 'migration'),
  ('Petugas Hibah / PSP', 'Akses khusus penyelesaian hibah dan PSP.',
   '["dashboard.view","grant.view","grant.manage","inventory.type.bmmn","search.view"]'::jsonb, true, true, 'migration'),
  ('Viewer', 'Akses baca tanpa perubahan data.',
   '["dashboard.view","inventory.view","inventory.type.btd","inventory.type.bdn","inventory.type.bmmn","reports.view","search.view"]'::jsonb, true, true, 'migration')
on conflict (name) do update set
  description = excluded.description,
  system = true,
  updated_at = now();

-- Parameter default kategori BDN.
insert into public.app_parameters (group_code, code, label, applies_to, active, system, sort_order, created_by)
values
  ('bdn_category','barang_lartas_ps_53_4','Barang Lartas Ps. 53 (4)','BDN',true,true,10,'migration'),
  ('bdn_category','barang_sarkut_ditegah','Barang/Sarkut yg Ditegah Pejabat BC','BDN',true,true,20,'migration'),
  ('bdn_category','barang_sarkut_ditinggalkan','Barang/Sarkut yg Ditinggalkan di KP','BDN',true,true,30,'migration'),
  ('bdn_category','bkc_pelanggar_tidak_dikenal','BKC & Barang Lain yg berasal dari Pelanggar Tidak Dikenal','BDN',true,true,40,'migration'),
  ('bdn_category','bkc_pemilik_tidak_diketahui','BKC yg berasal dari pemilik tidak diketahui','BDN',true,true,50,'migration')
on conflict (group_code, code) do update set
  label = excluded.label, applies_to = excluded.applies_to, system = true, updated_at = now();

-- Parameter default jenis barang.
insert into public.app_parameters (group_code, code, label, applies_to, active, system, sort_order, created_by)
values
  ('item_kind','barang_umum','Barang Umum','BTD,BDN,BMMN',true,true,10,'migration'),
  ('item_kind','barang_berbahaya_b3','Barang Berbahaya (B3)','BTD,BDN,BMMN',true,true,20,'migration'),
  ('item_kind','hewan_tumbuhan_hidup','Hewan atau Tumbuhan Hidup','BTD,BDN,BMMN',true,true,30,'migration'),
  ('item_kind','barang_peka_waktu','Barang Peka Waktu','BTD,BDN,BMMN',true,true,40,'migration'),
  ('item_kind','barang_berharga','Barang Berharga','BTD,BDN,BMMN',true,true,50,'migration')
on conflict (group_code, code) do update set
  label = excluded.label, applies_to = excluded.applies_to, system = true, updated_at = now();

-- Parameter default jenis pengeluaran.
insert into public.app_parameters (group_code, code, label, applies_to, active, system, sort_order, created_by)
values
  ('exit_type','pembatalan_bdn','PEMBATALAN BDN','BDN',true,true,10,'migration'),
  ('exit_type','impor_untuk_dipakai','IMPOR UTK DIPAKAI','BTD',true,true,20,'migration'),
  ('exit_type','reekspor','REEKSPOR','BTD,BDN',true,true,30,'migration'),
  ('exit_type','batal_ekspor','BATAL EKSPOR','BTD',true,true,40,'migration'),
  ('exit_type','ekspor','EKSPOR','BTD',true,true,50,'migration'),
  ('exit_type','keluarkan_ke_tpb','KELUARKAN KE TPB','BTD',true,true,60,'migration'),
  ('exit_type','lelang','LELANG','BTD,BDN,BMMN',true,true,70,'migration'),
  ('exit_type','musnah','MUSNAH','BTD,BDN,BMMN',true,true,80,'migration'),
  ('exit_type','psp','PSP','BTD,BDN,BMMN',true,true,90,'migration'),
  ('exit_type','hibah','HIBAH','BTD,BDN,BMMN',true,true,100,'migration'),
  ('exit_type','bmmn','BMMN','BTD,BDN',true,true,110,'migration'),
  ('exit_type','diserahkan_ke_ppns','DISERAHKAN KE PPNS','BDN',true,true,120,'migration'),
  ('exit_type','diserahkan_ke_aph_lain','DISERAHKAN KE APH LAIN','BTD,BDN',true,true,130,'migration'),
  ('exit_type','penghapusan','PENGHAPUSAN','BMMN',true,true,140,'migration')
on conflict (group_code, code) do update set
  label = excluded.label, applies_to = excluded.applies_to, system = true, updated_at = now();

-- Hapus validasi jenis barang yang sebelumnya hard-coded. Validasi baru membaca
-- parameter aktif sehingga admin dapat menambah pilihan tanpa migration baru.
alter table public.inventory_items
  drop constraint if exists inventory_item_kind_check;

create or replace function public.validate_livira_inventory_parameters()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  check_item_kind boolean := true;
  check_category boolean := true;
  check_exit_type boolean := true;
begin
  if tg_op = 'UPDATE' then
    check_item_kind := new.item_kind is distinct from old.item_kind;
    check_category := new.category is distinct from old.category or new.item_type is distinct from old.item_type;
    check_exit_type := new.exit_type is distinct from old.exit_type;
  end if;

  if check_item_kind and not exists (
       select 1
       from public.app_parameters p
       where p.group_code = 'item_kind'
         and p.active = true
         and p.label = new.item_kind
     ) then
    raise exception 'Jenis barang tidak aktif atau tidak terdaftar: %', new.item_kind
      using errcode = '23514';
  end if;

  if check_category and new.item_type = 'BDN'
     and not exists (
       select 1
       from public.app_parameters p
       where p.group_code = 'bdn_category'
         and p.active = true
         and p.label = new.category
     ) then
    raise exception 'Kategori BDN tidak aktif atau tidak terdaftar: %', new.category
      using errcode = '23514';
  end if;

  if check_exit_type and coalesce(new.exit_type, '') <> ''
     and not exists (
       select 1
       from public.app_parameters p
       where p.group_code = 'exit_type'
         and p.active = true
         and p.code = new.exit_type
         and new.item_type = any (string_to_array(replace(p.applies_to, ' ', ''), ','))
     ) then
    raise exception 'Jenis pengeluaran tidak aktif atau tidak sesuai dengan jenis inventory: %', new.exit_type
      using errcode = '23514';
  end if;

  return new;
end;
$$;

drop trigger if exists inventory_dynamic_parameters_insert_check on public.inventory_items;
create trigger inventory_dynamic_parameters_insert_check
before insert on public.inventory_items
for each row execute function public.validate_livira_inventory_parameters();

drop trigger if exists inventory_dynamic_parameters_update_check on public.inventory_items;
create trigger inventory_dynamic_parameters_update_check
before update of item_type, category, item_kind, exit_type on public.inventory_items
for each row execute function public.validate_livira_inventory_parameters();

alter table public.app_roles enable row level security;
alter table public.app_users enable row level security;
alter table public.app_parameters enable row level security;

revoke all on table public.app_roles from anon, authenticated;
revoke all on table public.app_users from anon, authenticated;
revoke all on table public.app_parameters from anon, authenticated;

grant usage on schema public to service_role;
grant all on table public.app_roles to service_role;
grant all on table public.app_users to service_role;
grant all on table public.app_parameters to service_role;

commit;

-- END MIGRATION: 007_access_approval_parameters.sql

-- ============================================================================
-- BEGIN MIGRATION: 008_idle_session_admin_delete.sql
-- ============================================================================
-- ================================================================
-- LIVIRA — IDLE SESSION & PENGHAPUSAN DATA OLEH ADMIN
-- Jalankan setelah migration 007.
--
-- Timeout 30 menit diterapkan pada aplikasi Go dan tidak membutuhkan
-- perubahan tabel. Migration ini menambahkan penghapusan inventory
-- yang atomik, sekaligus menyimpan snapshot audit sebelum data dihapus.
-- ================================================================

begin;

create table if not exists public.inventory_deletion_audit (
  id uuid primary key default gen_random_uuid(),
  inventory_id uuid not null,
  determination_no text not null default '',
  item_snapshot jsonb not null,
  disposition_snapshot jsonb not null default '[]'::jsonb,
  event_snapshot jsonb not null default '[]'::jsonb,
  deleted_by text not null,
  deleted_at timestamptz not null default now()
);

create index if not exists inventory_deletion_audit_time_idx
  on public.inventory_deletion_audit (deleted_at desc);

alter table public.inventory_deletion_audit enable row level security;
revoke all on table public.inventory_deletion_audit from anon, authenticated;

create or replace function public.admin_delete_inventory(
  p_inventory_id uuid,
  p_deleted_by text
)
returns void
language plpgsql
security definer
set search_path = public
as $$
declare
  v_item public.inventory_items%rowtype;
  v_dispositions jsonb;
  v_events jsonb;
begin
  select *
    into v_item
  from public.inventory_items
  where id = p_inventory_id
  for update;

  if not found then
    raise exception 'inventory_not_found';
  end if;

  select coalesce(jsonb_agg(to_jsonb(d) order by d.created_at), '[]'::jsonb)
    into v_dispositions
  from public.dispositions d
  where d.inventory_id = p_inventory_id;

  select coalesce(jsonb_agg(to_jsonb(e) order by e.created_at), '[]'::jsonb)
    into v_events
  from public.events e
  where e.inventory_id = p_inventory_id;

  insert into public.inventory_deletion_audit (
    inventory_id,
    determination_no,
    item_snapshot,
    disposition_snapshot,
    event_snapshot,
    deleted_by
  ) values (
    p_inventory_id,
    coalesce(v_item.determination_no, ''),
    to_jsonb(v_item),
    v_dispositions,
    v_events,
    coalesce(nullif(trim(p_deleted_by), ''), 'Administrator')
  );

  -- dispositions memakai ON DELETE RESTRICT terhadap inventory_items,
  -- sehingga proses harus dihapus lebih dahulu. Events terkait process
  -- ikut terhapus melalui cascade, lalu sisa events ikut cascade saat
  -- inventory dihapus.
  delete from public.dispositions where inventory_id = p_inventory_id;
  delete from public.inventory_items where id = p_inventory_id;
end;
$$;

revoke all on function public.admin_delete_inventory(uuid, text) from public, anon, authenticated;
grant execute on function public.admin_delete_inventory(uuid, text) to service_role;

commit;

-- END MIGRATION: 008_idle_session_admin_delete.sql

-- ============================================================================
-- BEGIN MIGRATION: 009_expand_system_parameters.sql
-- ============================================================================
-- =====================================================================
-- LIVIRA — PERLUAS PARAMETER DROPDOWN OPERASIONAL
-- Jalankan setelah 008_idle_session_admin_delete.sql.
-- Aman dijalankan ulang.
-- =====================================================================

begin;

-- Perluas kelompok parameter yang dapat dikelola administrator.
alter table public.app_parameters
  drop constraint if exists app_parameters_group_code_check;

alter table public.app_parameters
  add constraint app_parameters_group_code_check
  check (group_code in (
    'bdn_category',
    'item_kind',
    'unit',
    'allocation_purpose',
    'origin_tps',
    'load_type',
    'exit_type',
    'transfer_type'
  ));

-- Satuan barang.
insert into public.app_parameters
  (group_code, code, label, applies_to, active, system, sort_order, created_by)
values
  ('unit','ampoule','Ampoule','BTD,BDN,BMMN',true,true,10,'migration'),
  ('unit','bobbin','Bobbin','BTD,BDN,BMMN',true,true,20,'migration'),
  ('unit','bundle','Bundle','BTD,BDN,BMMN',true,true,30,'migration'),
  ('unit','bag','Bag','BTD,BDN,BMMN',true,true,40,'migration'),
  ('unit','bale','Bale','BTD,BDN,BMMN',true,true,50,'migration'),
  ('unit','barrel_petroleum_458_987_dm3','Barrel (petroleum) (458,987 dm3)','BTD,BDN,BMMN',true,true,60,'migration'),
  ('unit','bottle','Bottle','BTD,BDN,BMMN',true,true,70,'migration'),
  ('unit','box','Box','BTD,BDN,BMMN',true,true,80,'migration'),
  ('unit','can','Can','BTD,BDN,BMMN',true,true,90,'migration'),
  ('unit','coil','Coil','BTD,BDN,BMMN',true,true,100,'migration'),
  ('unit','centimetre','Centimetre','BTD,BDN,BMMN',true,true,110,'migration'),
  ('unit','crate','Crate','BTD,BDN,BMMN',true,true,120,'migration'),
  ('unit','case','Case','BTD,BDN,BMMN',true,true,130,'migration'),
  ('unit','carton','Carton','BTD,BDN,BMMN',true,true,140,'migration'),
  ('unit','drum','Drum','BTD,BDN,BMMN',true,true,150,'migration'),
  ('unit','dozen','Dozen','BTD,BDN,BMMN',true,true,160,'migration'),
  ('unit','gram','Gram','BTD,BDN,BMMN',true,true,170,'migration'),
  ('unit','kilogram','Kilogram','BTD,BDN,BMMN',true,true,180,'migration'),
  ('unit','litre_1_dm3','Litre ( 1 dm3 )','BTD,BDN,BMMN',true,true,190,'migration'),
  ('unit','milligram','Milligram','BTD,BDN,BMMN',true,true,200,'migration'),
  ('unit','millilitre','Millilitre','BTD,BDN,BMMN',true,true,210,'migration'),
  ('unit','millimetre','Millimetre','BTD,BDN,BMMN',true,true,220,'migration'),
  ('unit','square_metre','Square metre','BTD,BDN,BMMN',true,true,230,'migration'),
  ('unit','cubic_metre','Cubic metre','BTD,BDN,BMMN',true,true,240,'migration'),
  ('unit','metre','Metre','BTD,BDN,BMMN',true,true,250,'migration'),
  ('unit','unpacked_or_unpackaged','Unpacked or unpackaged','BTD,BDN,BMMN',true,true,260,'migration'),
  ('unit','number_of_international_units','Number of international units','BTD,BDN,BMMN',true,true,270,'migration'),
  ('unit','number_of_pairs','number of pairs','BTD,BDN,BMMN',true,true,280,'migration'),
  ('unit','piece','Piece','BTD,BDN,BMMN',true,true,290,'migration'),
  ('unit','pail','Pail','BTD,BDN,BMMN',true,true,300,'migration'),
  ('unit','tray_tray_pack','Tray / Tray Pack','BTD,BDN,BMMN',true,true,310,'migration'),
  ('unit','pallet','Pallet','BTD,BDN,BMMN',true,true,320,'migration'),
  ('unit','roll','Roll','BTD,BDN,BMMN',true,true,330,'migration'),
  ('unit','reel','Reel','BTD,BDN,BMMN',true,true,340,'migration'),
  ('unit','sack','Sack','BTD,BDN,BMMN',true,true,350,'migration'),
  ('unit','set','Set','BTD,BDN,BMMN',true,true,360,'migration'),
  ('unit','sheet','Sheet','BTD,BDN,BMMN',true,true,370,'migration'),
  ('unit','stick_cigarette','Stick, cigarette','BTD,BDN,BMMN',true,true,380,'migration'),
  ('unit','metric_ton_1000_kg','Metric ton (1000 kg)','BTD,BDN,BMMN',true,true,390,'migration'),
  ('unit','bulk_liquid','Bulk, liquid','BTD,BDN,BMMN',true,true,400,'migration'),
  ('unit','yard_0_9144_m','Yard (0.9144 m)','BTD,BDN,BMMN',true,true,410,'migration')
on conflict (group_code, code) do update set
  label = excluded.label,
  applies_to = excluded.applies_to,
  system = true,
  updated_at = now();

-- Jenis peruntukan BMMN.
insert into public.app_parameters
  (group_code, code, label, applies_to, active, system, sort_order, created_by)
values
  ('allocation_purpose','lelang','Lelang','BMMN',true,true,10,'migration'),
  ('allocation_purpose','musnah','Musnah','BMMN',true,true,20,'migration'),
  ('allocation_purpose','hibah','Hibah','BMMN',true,true,30,'migration'),
  ('allocation_purpose','psp','PSP','BMMN',true,true,40,'migration')
on conflict (group_code, code) do update set
  label = excluded.label,
  applies_to = excluded.applies_to,
  system = true,
  updated_at = now();

-- TPS asal.
insert into public.app_parameters
  (group_code, code, label, applies_to, active, system, sort_order, created_by)
values
  ('origin_tps','pt_agung_raya','PT Agung Raya','BTD,BDN',true,true,10,'migration'),
  ('origin_tps','pt_indonesian_air_marine_supply_utara','PT Indonesian Air & Marine Supply (Utara)','BTD,BDN',true,true,20,'migration'),
  ('origin_tps','pt_indonesian_air_marine_supply_barat','PT Indonesian Air & Marine Supply (Barat)','BTD,BDN',true,true,30,'migration'),
  ('origin_tps','pt_pelabuhan_indonesia_ii_persero','PT Pelabuhan Indonesia II (Persero)','BTD,BDN',true,true,40,'migration'),
  ('origin_tps','pt_indofood_sukses_makmur_tbk','PT Indofood Sukses Makmur Tbk','BTD,BDN',true,true,50,'migration'),
  ('origin_tps','pt_lautan_tirta_transportama','PT Lautan Tirta Transportama','BTD,BDN',true,true,60,'migration'),
  ('origin_tps','pt_multi_terminal_indonesia_cdc_banda','PT Multi Terminal Indonesia (CDC Banda)','BTD,BDN',true,true,70,'migration'),
  ('origin_tps','pt_pelabuhan_tanjung_priok_ambon','PT Pelabuhan Tanjung Priok (Ambon)','BTD,BDN',true,true,80,'migration'),
  ('origin_tps','pt_pelabuhan_tanjung_priok_101_101u','PT Pelabuhan Tanjung Priok (101-101U)','BTD,BDN',true,true,90,'migration'),
  ('origin_tps','pt_ipc_terminal_petikemas_terminal_3','PT IPC Terminal Petikemas (Terminal 3)','BTD,BDN',true,true,100,'migration'),
  ('origin_tps','pt_pelabuhan_indonesia_persero_regional_2_tanjung_priok','PT. Pelabuhan Indonesia (Persero) Regional 2 Tanjung Priok','BTD,BDN',true,true,110,'migration'),
  ('origin_tps','pt_primanata_jasa_persada','PT Primanata Jasa Persada','BTD,BDN',true,true,120,'migration'),
  ('origin_tps','pt_wira_mitra_prima','PT Wira Mitra Prima','BTD,BDN',true,true,130,'migration'),
  ('origin_tps','pt_pelabuhan_indonesia_ii_persero_npct1','PT Pelabuhan Indonesia II (Persero) (NPCT1)','BTD,BDN',true,true,140,'migration'),
  ('origin_tps','pt_pesaka_loka_kirana','PT Pesaka Loka Kirana','BTD,BDN',true,true,150,'migration'),
  ('origin_tps','pt_dharma_kartika_bhakti','PT Dharma Kartika Bhakti','BTD,BDN',true,true,160,'migration'),
  ('origin_tps','pt_inti_mandiri_utama_trans','PT Inti Mandiri Utama Trans','BTD,BDN',true,true,170,'migration'),
  ('origin_tps','pt_agung_raya_barat','PT Agung Raya (Barat)','BTD,BDN',true,true,180,'migration')
on conflict (group_code, code) do update set
  label = excluded.label,
  applies_to = excluded.applies_to,
  system = true,
  updated_at = now();

-- Jenis muatan dan jenis serah terima.
insert into public.app_parameters
  (group_code, code, label, applies_to, active, system, sort_order, created_by)
values
  ('load_type','FCL','FCL','BTD,BDN,BMMN',true,true,10,'migration'),
  ('load_type','LCL','LCL','BTD,BDN,BMMN',true,true,20,'migration'),
  ('transfer_type','hibah','Hibah','BMMN',true,true,10,'migration'),
  ('transfer_type','psp','PSP','BMMN',true,true,20,'migration')
on conflict (group_code, code) do update set
  label = excluded.label,
  applies_to = excluded.applies_to,
  system = true,
  updated_at = now();

-- FCL/LCL sebelumnya dibatasi oleh CHECK hard-coded. Setelah migration ini,
-- validasinya membaca parameter aktif pada app_parameters.
alter table public.inventory_items
  drop constraint if exists inventory_items_load_type_check;

create or replace function public.validate_livira_inventory_parameters()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  check_item_kind boolean := true;
  check_category boolean := true;
  check_unit boolean := true;
  check_origin_tps boolean := true;
  check_load_type boolean := true;
  check_allocation_purpose boolean := true;
  check_allocation_proposal boolean := true;
  check_allocation_approval boolean := true;
  check_exit_type boolean := true;
begin
  if tg_op = 'UPDATE' then
    check_item_kind := new.item_kind is distinct from old.item_kind;
    check_category := new.category is distinct from old.category or new.item_type is distinct from old.item_type;
    check_unit := new.unit is distinct from old.unit;
    check_origin_tps := new.origin_warehouse is distinct from old.origin_warehouse;
    check_load_type := new.load_type is distinct from old.load_type;
    check_allocation_purpose := new.allocation_purpose is distinct from old.allocation_purpose;
    check_allocation_proposal := new.allocation_proposal_type is distinct from old.allocation_proposal_type;
    check_allocation_approval := new.allocation_approval_type is distinct from old.allocation_approval_type;
    check_exit_type := new.exit_type is distinct from old.exit_type;
  end if;

  if check_item_kind and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'item_kind' and p.active = true and p.label = new.item_kind
     ) then
    raise exception 'Jenis barang tidak aktif atau tidak terdaftar: %', new.item_kind
      using errcode = '23514';
  end if;

  if check_unit and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'unit' and p.active = true and p.label = new.unit
     ) then
    raise exception 'Satuan tidak aktif atau tidak terdaftar: %', new.unit
      using errcode = '23514';
  end if;

  if check_origin_tps and coalesce(new.origin_warehouse, '') <> ''
     and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'origin_tps' and p.active = true and p.label = new.origin_warehouse
     ) then
    raise exception 'TPS asal tidak aktif atau tidak terdaftar: %', new.origin_warehouse
      using errcode = '23514';
  end if;

  if check_load_type and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'load_type' and p.active = true and p.code = new.load_type
     ) then
    raise exception 'Jenis muatan tidak aktif atau tidak terdaftar: %', new.load_type
      using errcode = '23514';
  end if;

  if check_category and new.item_type = 'BDN'
     and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'bdn_category' and p.active = true and p.label = new.category
     ) then
    raise exception 'Kategori BDN tidak aktif atau tidak terdaftar: %', new.category
      using errcode = '23514';
  end if;

  if check_allocation_purpose and coalesce(new.allocation_purpose, '') <> ''
     and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'allocation_purpose' and p.active = true and p.label = new.allocation_purpose
     ) then
    raise exception 'Jenis peruntukan tidak aktif atau tidak terdaftar: %', new.allocation_purpose
      using errcode = '23514';
  end if;

  if check_allocation_proposal and coalesce(new.allocation_proposal_type, '') <> ''
     and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'allocation_purpose' and p.active = true and p.label = new.allocation_proposal_type
     ) then
    raise exception 'Jenis usulan peruntukan tidak aktif atau tidak terdaftar: %', new.allocation_proposal_type
      using errcode = '23514';
  end if;

  if check_allocation_approval and coalesce(new.allocation_approval_type, '') <> ''
     and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'allocation_purpose' and p.active = true and p.label = new.allocation_approval_type
     ) then
    raise exception 'Jenis persetujuan peruntukan tidak aktif atau tidak terdaftar: %', new.allocation_approval_type
      using errcode = '23514';
  end if;

  if check_exit_type and coalesce(new.exit_type, '') <> ''
     and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'exit_type'
         and p.active = true
         and p.code = new.exit_type
         and new.item_type = any (string_to_array(replace(p.applies_to, ' ', ''), ','))
     ) then
    raise exception 'Jenis pengeluaran tidak aktif atau tidak sesuai dengan jenis inventory: %', new.exit_type
      using errcode = '23514';
  end if;

  return new;
end;
$$;

drop trigger if exists inventory_dynamic_parameters_update_check on public.inventory_items;
create trigger inventory_dynamic_parameters_update_check
before update of item_type, category, item_kind, unit, origin_warehouse, load_type,
  allocation_purpose, allocation_proposal_type, allocation_approval_type, exit_type
on public.inventory_items
for each row execute function public.validate_livira_inventory_parameters();

create or replace function public.validate_livira_disposition_parameters()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
begin
  if coalesce(new.transfer_type, '') <> ''
     and not exists (
       select 1 from public.app_parameters p
       where p.group_code = 'transfer_type'
         and p.active = true
         and p.code = new.transfer_type
     ) then
    raise exception 'Jenis serah terima tidak aktif atau tidak terdaftar: %', new.transfer_type
      using errcode = '23514';
  end if;
  return new;
end;
$$;

drop trigger if exists disposition_dynamic_parameters_check on public.dispositions;
create trigger disposition_dynamic_parameters_check
before insert or update of transfer_type on public.dispositions
for each row execute function public.validate_livira_disposition_parameters();

commit;

-- END MIGRATION: 009_expand_system_parameters.sql

-- ============================================================================
-- BEGIN MIGRATION: 010_capacity_multi_container_dashboard.sql
-- ============================================================================
-- =====================================================================
-- LIVIRA — KAPASITAS YOR/SOR, MULTI KONTAINER, DAN DASHBOARD POPUP
-- Jalankan setelah 009_expand_system_parameters.sql.
-- Aman dijalankan ulang.
-- =====================================================================

begin;

-- Kapasitas YOR dinyatakan dalam TEU (ekuivalen peti kemas 20 kaki),
-- sedangkan kapasitas SOR dinyatakan dalam meter kubik. Numeric dipakai
-- agar peti kemas 45 kaki (2,25 TEU) dan volume desimal dapat dicatat.
alter table public.facilities
  drop constraint if exists facilities_yard_capacity_check,
  drop constraint if exists facilities_yard_used_check,
  drop constraint if exists facilities_shed_capacity_check,
  drop constraint if exists facilities_shed_used_check;

alter table public.facilities
  alter column yard_capacity type numeric(14,2) using yard_capacity::numeric,
  alter column yard_used type numeric(14,2) using yard_used::numeric,
  alter column shed_capacity type numeric(14,2) using shed_capacity::numeric,
  alter column shed_used type numeric(14,2) using shed_used::numeric;

alter table public.facilities
  add constraint facilities_yard_capacity_check check (yard_capacity >= 0),
  add constraint facilities_yard_used_check check (yard_used >= 0),
  add constraint facilities_shed_capacity_check check (shed_capacity >= 0),
  add constraint facilities_shed_used_check check (shed_used >= 0);

-- Data FCL menyimpan ukuran tiap kontainer. Data LCL menyimpan perkiraan
-- volume barang. Satu nomor penetapan dapat menghasilkan beberapa baris
-- inventory dengan reference_no berbeda dan determination_no yang sama.
alter table public.inventory_items
  add column if not exists container_size text not null default '',
  add column if not exists estimated_volume_m3 numeric(14,2) not null default 0;

-- Normalisasi data yang sudah ada agar kompatibel dengan struktur baru.
update public.inventory_items
set container_size = '20'
where upper(coalesce(load_type, '')) = 'FCL'
  and coalesce(container_no, '') <> ''
  and coalesce(container_size, '') = '';

update public.inventory_items
set container_no = '',
    container_size = '',
    estimated_volume_m3 = greatest(
      coalesce(nullif(estimated_volume_m3, 0), quantity, 0.01),
      0.01
    )
where upper(coalesce(load_type, '')) = 'LCL';

alter table public.inventory_items
  drop constraint if exists inventory_container_size_check,
  drop constraint if exists inventory_estimated_volume_check,
  drop constraint if exists inventory_fcl_container_detail_check,
  drop constraint if exists inventory_lcl_volume_check;

alter table public.inventory_items
  add constraint inventory_container_size_check
    check (container_size in ('', '20', '40', '45')),
  add constraint inventory_estimated_volume_check
    check (estimated_volume_m3 >= 0),
  add constraint inventory_fcl_container_detail_check
    check (
      upper(load_type) <> 'FCL'
      or container_no = ''
      or container_size in ('20', '40', '45')
    ),
  add constraint inventory_lcl_volume_check
    check (
      upper(load_type) <> 'LCL'
      or (
        estimated_volume_m3 > 0
        and container_no = ''
        and container_size = ''
      )
    );

-- Cegah satu kontainer aktif tercatat dua kali. Nomor dinormalisasi dengan
-- mengabaikan spasi, tanda hubung, dan perbedaan huruf besar/kecil.
create unique index if not exists inventory_active_container_unique_idx
  on public.inventory_items (
    upper(regexp_replace(container_no, '[^A-Za-z0-9]', '', 'g'))
  )
  where is_active = true and coalesce(container_no, '') <> '';

comment on column public.facilities.yard_capacity is
  'Kapasitas YOR dalam TEU (ekuivalen peti kemas 20 kaki).';
comment on column public.facilities.shed_capacity is
  'Kapasitas SOR/gudang dalam meter kubik.';
comment on column public.inventory_items.container_size is
  'Ukuran kontainer FCL: 20, 40, atau 45 kaki.';
comment on column public.inventory_items.estimated_volume_m3 is
  'Perkiraan volume barang LCL dalam meter kubik.';

commit;

-- END MIGRATION: 010_capacity_multi_container_dashboard.sql

-- ============================================================================
-- BEGIN MIGRATION: 011_container_size_options_ui.sql
-- ============================================================================
-- =====================================================================
-- LIVIRA — OPSI UKURAN PETI KEMAS 20', 40', 40' HC, 45' HC
-- Jalankan setelah migration kapasitas/multi-kontainer.
-- Aman dijalankan ulang.
-- =====================================================================

begin;

alter table public.inventory_items
  add column if not exists container_size text not null default '';

-- Konversi kode lama 45 menjadi 45HC agar label dan perhitungan konsisten.
update public.inventory_items
set container_size = '45HC'
where upper(trim(coalesce(container_size, ''))) = '45';

alter table public.inventory_items
  drop constraint if exists inventory_container_size_check,
  drop constraint if exists inventory_fcl_container_detail_check;

alter table public.inventory_items
  add constraint inventory_container_size_check
    check (upper(container_size) in ('', '20', '40', '40HC', '45HC')),
  add constraint inventory_fcl_container_detail_check
    check (
      upper(load_type) <> 'FCL'
      or container_no = ''
      or upper(container_size) in ('20', '40', '40HC', '45HC')
    );

comment on column public.inventory_items.container_size is
  'Ukuran peti kemas FCL: 20, 40, 40HC, atau 45HC.';

commit;

-- END MIGRATION: 011_container_size_options_ui.sql

-- ============================================================================
-- BEGIN MIGRATION: 012_reporting_pagination_multi_goods_pfpd.sql
-- ============================================================================
-- =====================================================================
-- LIVIRA — PAGINASI, MULTI URAIAN PER KONTAINER, DAN PENELITIAN PFPD
-- Jalankan setelah 011_container_size_options_ui.sql.
-- Aman dijalankan ulang.
-- =====================================================================

begin;

alter table public.inventory_items
  add column if not exists physical_unit_id text not null default '',
  add column if not exists occupancy_primary boolean not null default true,
  add column if not exists pfpd_required boolean not null default true;

update public.inventory_items
set physical_unit_id = id::text
where coalesce(trim(physical_unit_id), '') = '';

update public.inventory_items
set pfpd_required = true
where research_request_no <> ''
   or hs_code <> ''
   or status_code in ('request_penelitian_pfpd', 'penelitian_pfpd');

-- Satu kontainer dapat memiliki beberapa baris uraian barang. Hanya satu
-- baris utama yang mewakili unit fisik untuk perhitungan kapasitas YOR/SOR.
drop index if exists public.inventory_active_container_unique_idx;

create unique index if not exists inventory_active_container_primary_unique_idx
  on public.inventory_items (
    upper(regexp_replace(container_no, '[^A-Za-z0-9]', '', 'g'))
  )
  where is_active = true
    and occupancy_primary = true
    and coalesce(container_no, '') <> '';

create index if not exists inventory_physical_unit_idx
  on public.inventory_items (physical_unit_id, occupancy_primary, is_active);

create index if not exists inventory_research_request_idx
  on public.inventory_items (research_request_no, status_code, is_active)
  where research_request_no <> '';

comment on column public.inventory_items.physical_unit_id is
  'Identitas unit fisik bersama untuk beberapa uraian barang dalam kontainer/LCL yang sama.';
comment on column public.inventory_items.occupancy_primary is
  'Hanya baris utama yang dihitung dalam kapasitas YOR/SOR.';
comment on column public.inventory_items.pfpd_required is
  'Menandai apakah hasil pencacahan memerlukan penelitian PFPD.';

commit;

-- END MIGRATION: 012_reporting_pagination_multi_goods_pfpd.sql

-- ============================================================================
-- BEGIN MIGRATION: 013_titipan_rekonsiliasi_lelang_dashboard.sql
-- ============================================================================
-- =====================================================================
-- LIVIRA — BARANG TITIPAN, REKONSILIASI, DAN PENYELESAIAN LELANG
-- Jalankan setelah 012_reporting_pagination_multi_goods_pfpd.sql.
-- Aman dijalankan ulang.
-- =====================================================================

begin;

alter table public.inventory_items
  add column if not exists entrusted_category text not null default '',
  add column if not exists source_office text not null default '';

alter table public.inventory_items drop constraint if exists inventory_items_item_type_check;
alter table public.inventory_items
  add constraint inventory_items_item_type_check
  check (item_type in ('BTD','BDN','BMMN','TITIPAN'));

alter table public.inventory_items drop constraint if exists inventory_items_origin_type_check;
alter table public.inventory_items
  add constraint inventory_items_origin_type_check
  check (origin_type in ('BTD','BDN','BMMN','TITIPAN'));

alter table public.inventory_items drop constraint if exists inventory_entrusted_category_check;
alter table public.inventory_items
  add constraint inventory_entrusted_category_check
  check (
    (item_type <> 'TITIPAN' and entrusted_category = '') or
    (item_type = 'TITIPAN' and entrusted_category in ('BTD','BDN','BMMN','Tidak Teridentifikasi'))
  );

alter table public.inventory_items drop constraint if exists inventory_entrusted_source_office_check;
alter table public.inventory_items
  add constraint inventory_entrusted_source_office_check
  check (item_type <> 'TITIPAN' or btrim(source_office) <> '');

alter table public.dispositions
  add column if not exists schedule_document_no text not null default '',
  add column if not exists schedule_document_date timestamptz;

create index if not exists disposition_schedule_document_idx
  on public.dispositions (disposition_type, schedule_document_no, status_code, is_active)
  where schedule_document_no <> '';

create table if not exists public.reconciliations (
  id uuid primary key default gen_random_uuid(),
  reconciliation_type text not null check (reconciliation_type in ('recorded_not_found','found_not_recorded')),
  action text not null check (action in ('removed','added')),
  inventory_id uuid references public.inventory_items(id) on delete set null,
  inventory_reference text not null default '',
  inventory_type text not null check (inventory_type in ('BTD','BDN','BMMN','TITIPAN')),
  previous_status_code text not null default '',
  previous_status_label text not null default '',
  result_status_code text not null default '',
  result_status_label text not null default '',
  notes text not null,
  actor text not null,
  created_at timestamptz not null default now()
);

create index if not exists reconciliations_created_idx
  on public.reconciliations (created_at desc);
create index if not exists reconciliations_inventory_idx
  on public.reconciliations (inventory_id, created_at desc);

alter table public.reconciliations enable row level security;
revoke all on table public.reconciliations from anon, authenticated;
grant all on table public.reconciliations to service_role;

-- Kolom status lokasi langsung menampilkan nama TPS/TPP tanpa frasa tambahan.
update public.inventory_items
set location_status = case
  when at_tpp then coalesce(nullif(btrim(facility_name), ''), nullif(btrim(location), ''), 'TPP belum ditentukan')
  when item_type = 'TITIPAN' then coalesce(nullif(btrim(source_office), ''), nullif(btrim(location), ''), 'Kantor/unit belum ditentukan')
  else coalesce(nullif(btrim(origin_warehouse), ''), nullif(btrim(location), ''), 'TPS belum ditentukan')
end;

-- Isi identitas ND jadwal untuk data lama berdasarkan event penjadwalan terakhir.
update public.dispositions d
set schedule_document_no = coalesce((
      select e.document_no
      from public.events e
      where e.disposition_id = d.id and e.code = 'jadwal_lelang'
      order by e.created_at desc
      limit 1
    ), ''),
    schedule_document_date = (
      select e.document_date
      from public.events e
      where e.disposition_id = d.id and e.code = 'jadwal_lelang'
      order by e.created_at desc
      limit 1
    )
where d.schedule_document_no = ''
  and exists (
    select 1 from public.events e
    where e.disposition_id = d.id and e.code = 'jadwal_lelang' and coalesce(e.document_no, '') <> ''
  );

-- Tambahkan hak akses baru ke role inventory bawaan. Role lain tetap dapat
-- dikustomisasi melalui menu Role & Hak Akses.
update public.app_roles
set permissions = (
  select jsonb_agg(value order by value)
  from (
    select distinct value
    from (
      select value from jsonb_array_elements_text(coalesce(permissions, '[]'::jsonb)) value
      union all select 'inventory.type.titipan'
      union all select 'reconciliation.view'
      union all select 'reconciliation.manage'
    ) combined
  ) p
), updated_at = now()
where lower(name) = 'petugas inventory';

update public.app_roles
set permissions = (
  select jsonb_agg(value order by value)
  from (
    select distinct value
    from (
      select value from jsonb_array_elements_text(coalesce(permissions, '[]'::jsonb)) value
      union all select 'inventory.type.titipan'
      union all select 'reconciliation.view'
    ) combined
  ) p
), updated_at = now()
where lower(name) = 'viewer';

-- Barang titipan memakai master jenis barang, satuan, dan jenis muatan yang sama.
update public.app_parameters
set applies_to = case
  when trim(applies_to) = '' then 'BTD,BDN,BMMN,TITIPAN'
  when not ('TITIPAN' = any (string_to_array(replace(applies_to, ' ', ''), ','))) then applies_to || ',TITIPAN'
  else applies_to
end,
updated_at = now()
where group_code in ('item_kind','unit','load_type');

-- Sesuaikan parameter jenis pengeluaran dengan matriks terbaru.
delete from public.app_parameters
where group_code = 'exit_type';

insert into public.app_parameters (group_code, code, label, applies_to, active, system, sort_order, created_by)
values
  ('exit_type','impor_untuk_dipakai','IMPOR UTK DIPAKAI','BTD',true,true,10,'migration'),
  ('exit_type','reekspor','REEKSPOR','BTD,BDN',true,true,20,'migration'),
  ('exit_type','batal_ekspor','BATAL EKSPOR','BTD',true,true,30,'migration'),
  ('exit_type','ekspor','EKSPOR','BTD',true,true,40,'migration'),
  ('exit_type','keluarkan_ke_tpb','KELUARKAN KE TPB','BTD',true,true,50,'migration'),
  ('exit_type','lelang','LELANG','BTD,BDN,BMMN',true,true,60,'migration'),
  ('exit_type','musnah','MUSNAH','BTD,BDN,BMMN',true,true,70,'migration'),
  ('exit_type','psp','PSP','BTD,BDN,BMMN',true,true,80,'migration'),
  ('exit_type','hibah','HIBAH','BTD,BDN,BMMN',true,true,90,'migration'),
  ('exit_type','diserahkan_ke_aph_lain','DISERAHKAN KE APH LAIN','BTD,BDN',true,true,100,'migration'),
  ('exit_type','pembatalan_bdn','PEMBATALAN BDN','BDN',true,true,110,'migration'),
  ('exit_type','diserahkan_ke_ppns','DISERAHKAN KE PPNS','BDN',true,true,120,'migration'),
  ('exit_type','penghapusan','PENGHAPUSAN','BMMN',true,true,130,'migration'),
  ('exit_type','pengeluaran_barang_titipan','PENGELUARAN BARANG TITIPAN','TITIPAN',true,true,140,'migration')
on conflict (group_code, code) do update set
  label = excluded.label,
  applies_to = excluded.applies_to,
  active = true,
  sort_order = excluded.sort_order,
  updated_at = now();

commit;

-- END MIGRATION: 013_titipan_rekonsiliasi_lelang_dashboard.sql

-- ============================================================================
-- BEGIN MIGRATION: 014_multi_barang_kondisi_htl_per_item.sql
-- ============================================================================
-- =====================================================================
-- LIVIRA — MULTI BARANG AWAL, KONDISI BARANG, DAN HTL PER ITEM
-- Jalankan setelah 013_titipan_rekonsiliasi_lelang_dashboard.sql.
-- Aman dijalankan ulang. Migration ini tidak menghapus data inventory.
-- =====================================================================

begin;

-- Kondisi barang menjadi master parameter yang dapat dikelola admin.
alter table public.app_parameters
  drop constraint if exists app_parameters_group_code_check;

alter table public.app_parameters
  add constraint app_parameters_group_code_check
  check (group_code in (
    'bdn_category',
    'item_kind',
    'goods_condition',
    'unit',
    'allocation_purpose',
    'origin_tps',
    'load_type',
    'exit_type',
    'transfer_type'
  ));

insert into public.app_parameters
  (group_code, code, label, applies_to, active, system, sort_order, created_by)
values
  ('goods_condition','baru','Baru','BTD,BDN,BMMN,TITIPAN',true,true,10,'migration'),
  ('goods_condition','bekas','Bekas','BTD,BDN,BMMN,TITIPAN',true,true,20,'migration'),
  ('goods_condition','rusak','Rusak','BTD,BDN,BMMN,TITIPAN',true,true,30,'migration'),
  ('goods_condition','segar','Segar','BTD,BDN,BMMN,TITIPAN',true,true,40,'migration'),
  ('goods_condition','busuk','Busuk','BTD,BDN,BMMN,TITIPAN',true,true,50,'migration')
on conflict (group_code, code) do update set
  label = excluded.label,
  applies_to = excluded.applies_to,
  system = true,
  updated_at = now();

alter table public.inventory_items
  add column if not exists goods_condition text not null default '';

-- Nilai tidak dikunci dengan CHECK hard-coded agar admin dapat menambah
-- parameter kondisi baru. Validasi dilakukan terhadap app_parameters aktif.
alter table public.inventory_items
  drop constraint if exists inventory_goods_condition_check;

create or replace function public.validate_livira_goods_condition()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
begin
  if coalesce(trim(new.goods_condition), '') <> ''
     and not exists (
       select 1
       from public.app_parameters p
       where p.group_code = 'goods_condition'
         and p.active = true
         and p.label = new.goods_condition
         and new.item_type = any (
           string_to_array(replace(p.applies_to, ' ', ''), ',')
         )
     ) then
    raise exception 'Kondisi barang tidak aktif, tidak terdaftar, atau tidak sesuai jenis inventory: %', new.goods_condition
      using errcode = '23514';
  end if;
  return new;
end;
$$;

drop trigger if exists inventory_goods_condition_parameter_check
  on public.inventory_items;
create trigger inventory_goods_condition_parameter_check
before insert or update of goods_condition, item_type
on public.inventory_items
for each row execute function public.validate_livira_goods_condition();

create index if not exists inventory_goods_condition_idx
  on public.inventory_items (goods_condition, is_active, updated_at desc)
  where goods_condition <> '';

comment on column public.inventory_items.goods_condition is
  'Kondisi fisik barang hasil pencacahan. Nilai mengikuti parameter aktif pada kelompok goods_condition.';

commit;

-- END MIGRATION: 014_multi_barang_kondisi_htl_per_item.sql

-- ============================================================================
-- BEGIN MIGRATION: 015_document_upload_admin_search_access.sql
-- ============================================================================
-- ================================================================
-- LIVIRA — DOKUMEN ACTION, AKSES KAPASITAS, NORMALISASI LELANG, DAN INDEKS KINERJA
-- Jalankan setelah migration 014.
-- ================================================================

begin;

create table if not exists public.uploaded_documents (
  id uuid primary key default gen_random_uuid(),
  file_name text not null,
  mime_type text not null check (mime_type in (
    'application/pdf',
    'image/jpeg',
    'image/png',
    'image/webp',
    'image/gif'
  )),
  size_bytes bigint not null check (size_bytes > 0 and size_bytes <= 8388608),
  content_base64 text not null,
  uploaded_by text not null default '',
  created_at timestamptz not null default now()
);

alter table public.events
  add column if not exists document_id uuid references public.uploaded_documents(id) on delete set null;

create index if not exists events_document_id_idx
  on public.events (document_id)
  where document_id is not null;

create index if not exists uploaded_documents_created_at_idx
  on public.uploaded_documents (created_at desc);

comment on table public.uploaded_documents is
  'Dokumen PDF/gambar opsional yang dilampirkan pada penetapan dan action. Konten hanya diakses oleh backend service role.';
comment on column public.events.document_id is
  'Referensi dokumen pendukung yang dapat diunduh dari jejak audit timeline.';

alter table public.uploaded_documents enable row level security;
revoke all on table public.uploaded_documents from anon, authenticated;
grant all on table public.uploaded_documents to service_role;

-- Bea Cukai Tanjung Priok tidak mencatat komponen biaya pada proses lelang.
-- Kolom lama dipertahankan untuk kompatibilitas skema, tetapi seluruh nilainya dinormalkan ke nol.
update public.dispositions
set auction_cost = 0,
    updated_at = now()
where coalesce(auction_cost, 0) <> 0;

-- Role bawaan Petugas Inventory memperoleh akses pengelolaan kapasitas YOR/SOR.
-- Role kustom lain dapat diberikan hak yang sama melalui menu Role & Hak Akses.
-- Indeks pendukung dashboard dan ekspor performa kinerja. Tidak ada tabel
-- agregat baru: hasil tetap dihitung dari jejak audit event agar selalu konsisten.
create index if not exists events_performance_code_document_date_idx
  on public.events (code, document_date desc);

create index if not exists events_performance_inventory_created_idx
  on public.events (inventory_id, created_at asc);

create index if not exists inventory_origin_document_date_idx
  on public.inventory_items (origin_document_date)
  where origin_document_date is not null;

update public.app_roles
set permissions = (
  select jsonb_agg(value order by value)
  from (
    select distinct value
    from (
      select value
      from jsonb_array_elements_text(coalesce(permissions, '[]'::jsonb)) value
      union all
      select 'dashboard.capacity.manage'
    ) combined
  ) normalized
), updated_at = now()
where lower(name) = 'petugas inventory';

commit;

-- END MIGRATION: 015_document_upload_admin_search_access.sql

-- ============================================================================
-- BEGIN MIGRATION: 016_security_performance_hardening.sql
-- ============================================================================
-- =====================================================================
-- LIVIRA — PENGUATAN KEAMANAN, TRANSAKSI, DOKUMEN, DAN PERFORMA
-- Jalankan HANYA setelah migration 015_document_upload_admin_search_access.sql.
-- Aman dijalankan ulang. Migration 015 tidak perlu dijalankan kembali.
-- =====================================================================

begin;

create extension if not exists pg_trgm;

-- ---------------------------------------------------------------------
-- 1. Pencabutan sesi otomatis saat role/status/verifikasi berubah.
-- ---------------------------------------------------------------------
alter table public.app_users
  add column if not exists session_version bigint not null default 1;

create index if not exists app_users_session_version_idx
  on public.app_users (auth_user_id, session_version, approval_status);

create or replace function public.livira_bump_user_session_version()
returns trigger
language plpgsql
set search_path = public, pg_temp
as $$
begin
  if new.role_id is distinct from old.role_id
     or new.approval_status is distinct from old.approval_status
     or new.email_verified is distinct from old.email_verified then
    new.session_version := greatest(coalesce(old.session_version, 1), 1) + 1;
  end if;
  return new;
end;
$$;

drop trigger if exists app_users_bump_session_version on public.app_users;
create trigger app_users_bump_session_version
before update of role_id, approval_status, email_verified
on public.app_users
for each row execute function public.livira_bump_user_session_version();

create or replace function public.livira_revoke_role_sessions()
returns trigger
language plpgsql
set search_path = public, pg_temp
as $$
begin
  if new.permissions is distinct from old.permissions
     or new.active is distinct from old.active then
    update public.app_users
    set session_version = session_version + 1,
        updated_at = now()
    where role_id = new.id;
  end if;
  return new;
end;
$$;

drop trigger if exists app_roles_revoke_sessions on public.app_roles;
create trigger app_roles_revoke_sessions
after update of permissions, active
on public.app_roles
for each row execute function public.livira_revoke_role_sessions();

-- ---------------------------------------------------------------------
-- 2. Dokumen baru disimpan di private Supabase Storage.
--    Dokumen lama Base64 tetap dapat dibaca untuk kompatibilitas.
-- ---------------------------------------------------------------------
alter table public.uploaded_documents
  alter column content_base64 drop not null,
  add column if not exists storage_bucket text,
  add column if not exists storage_path text,
  add column if not exists sha256 text;

alter table public.uploaded_documents
  drop constraint if exists uploaded_documents_content_location_check;
alter table public.uploaded_documents
  add constraint uploaded_documents_content_location_check check (
    coalesce(length(content_base64), 0) > 0
    or (
      coalesce(length(storage_bucket), 0) > 0
      and coalesce(length(storage_path), 0) > 0
    )
  );

alter table public.uploaded_documents
  drop constraint if exists uploaded_documents_sha256_check;
alter table public.uploaded_documents
  add constraint uploaded_documents_sha256_check check (
    sha256 is null or sha256 ~ '^[0-9a-fA-F]{64}$'
  );

create unique index if not exists uploaded_documents_storage_object_uidx
  on public.uploaded_documents (storage_bucket, storage_path)
  where storage_bucket is not null and storage_path is not null;

create index if not exists uploaded_documents_sha256_idx
  on public.uploaded_documents (sha256)
  where sha256 is not null;

insert into storage.buckets (id, name, public, file_size_limit, allowed_mime_types)
values (
  'livira-documents',
  'livira-documents',
  false,
  8388608,
  array['application/pdf','image/jpeg','image/png','image/webp','image/gif']::text[]
)
on conflict (id) do update set
  public = false,
  file_size_limit = excluded.file_size_limit,
  allowed_mime_types = excluded.allowed_mime_types;

-- ---------------------------------------------------------------------
-- 3. Audit keamanan append-only.
-- ---------------------------------------------------------------------
create table if not exists public.audit_logs (
  id uuid primary key default gen_random_uuid(),
  actor_subject text not null default '',
  actor_name text not null default '',
  action text not null,
  entity_type text not null default '',
  entity_id text not null default '',
  outcome text not null default 'success'
    check (outcome in ('success','failed','denied')),
  ip_address text not null default '',
  user_agent text not null default '',
  request_id text not null default '',
  metadata jsonb not null default '{}'::jsonb
    check (jsonb_typeof(metadata) = 'object'),
  created_at timestamptz not null default now()
);

create index if not exists audit_logs_created_idx
  on public.audit_logs (created_at desc);
create index if not exists audit_logs_actor_idx
  on public.audit_logs (actor_subject, created_at desc);
create index if not exists audit_logs_entity_idx
  on public.audit_logs (entity_type, entity_id, created_at desc);
create index if not exists audit_logs_action_outcome_idx
  on public.audit_logs (action, outcome, created_at desc);

alter table public.audit_logs enable row level security;
revoke all on table public.audit_logs from anon, authenticated;
grant select, insert on table public.audit_logs to service_role;

-- ---------------------------------------------------------------------
-- 4. Satu kolom pencarian terindeks menggantikan OR ILIKE puluhan kolom.
-- ---------------------------------------------------------------------
alter table public.inventory_items
  add column if not exists search_text text not null default '';

create or replace function public.livira_inventory_search_text(i public.inventory_items)
returns text
language sql
immutable
set search_path = public, pg_temp
as $$
  select lower(concat_ws(' ',
    i.reference_no, i.item_type, i.origin_type,
    i.manifest_no, i.manifest_position,
    i.determination_no, i.category, i.entrusted_category, i.source_office,
    i.description, i.item_kind, i.goods_condition,
    i.quantity::text, i.unit, i.goods_value::text,
    i.location, i.location_status, i.owner_name, i.owner_address,
    i.origin_warehouse, i.facility_id, i.facility_name,
    i.load_type, i.container_no, i.container_size,
    i.estimated_volume_m3::text, i.physical_unit_id,
    i.research_request_no, i.hs_code, i.restriction_rule,
    i.origin_document_type, i.origin_document_no,
    i.allocation_purpose, i.allocation_proposal_type,
    i.allocation_proposal_no, i.allocation_approval_type,
    i.allocation_approval_no, i.exit_document_no, i.exit_type,
    i.exit_notes, i.status_code, i.status_label,
    i.current_disposition
  ));
$$;

create or replace function public.livira_set_inventory_search_text()
returns trigger
language plpgsql
set search_path = public, pg_temp
as $$
begin
  new.search_text := public.livira_inventory_search_text(new);
  return new;
end;
$$;

drop trigger if exists inventory_set_search_text on public.inventory_items;
create trigger inventory_set_search_text
before insert or update
on public.inventory_items
for each row execute function public.livira_set_inventory_search_text();

update public.inventory_items i
set search_text = public.livira_inventory_search_text(i)
where search_text is distinct from public.livira_inventory_search_text(i);

create index if not exists inventory_search_text_trgm_idx
  on public.inventory_items using gin (search_text gin_trgm_ops);

create index if not exists inventory_active_page_idx
  on public.inventory_items (is_active, updated_at desc, id);
create index if not exists inventory_type_active_page_idx
  on public.inventory_items (item_type, is_active, updated_at desc, id);
create index if not exists inventory_facility_active_page_idx
  on public.inventory_items (facility_id, is_active, updated_at desc, id);

-- Trigger kondisi barang lama diperketat agar perubahan status yang tidak
-- menyentuh kondisi barang tidak memvalidasi ulang nilai historis.
create or replace function public.validate_livira_goods_condition()
returns trigger
language plpgsql
security definer
set search_path = public, pg_temp
as $$
begin
  if tg_op = 'UPDATE'
     and new.goods_condition is not distinct from old.goods_condition
     and new.item_type is not distinct from old.item_type then
    return new;
  end if;
  if coalesce(trim(new.goods_condition), '') <> ''
     and not exists (
       select 1
       from public.app_parameters p
       where p.group_code = 'goods_condition'
         and p.active = true
         and p.label = new.goods_condition
         and new.item_type = any (
           string_to_array(replace(p.applies_to, ' ', ''), ',')
         )
     ) then
    raise exception 'Kondisi barang tidak aktif, tidak terdaftar, atau tidak sesuai jenis inventory: %', new.goods_condition
      using errcode = '23514';
  end if;
  return new;
end;
$$;

-- View akses pengguna menggabungkan akun dan role dalam satu query.
create or replace view public.app_user_access
with (security_invoker = true)
as
select
  u.*,
  coalesce(case when r.active then r.name end, '') as role_name,
  coalesce(case when r.active then r.permissions end, '[]'::jsonb) as permissions
from public.app_users u
left join public.app_roles r on r.id = u.role_id;

revoke all on public.app_user_access from anon, authenticated;
grant select on public.app_user_access to service_role;

-- ---------------------------------------------------------------------
-- 5. View proses dengan inventory tertanam: menghapus pola N+1 request.
-- ---------------------------------------------------------------------
create or replace view public.disposition_details
with (security_invoker = true)
as
select
  d.*,
  to_jsonb(i) - 'search_text' as inventory,
  i.is_active as inventory_is_active,
  i.facility_id as inventory_facility_id,
  i.item_type as inventory_item_type,
  i.determination_date as inventory_determination_date,
  i.goods_value as inventory_goods_value,
  i.search_text as inventory_search_text
from public.dispositions d
join public.inventory_items i on i.id = d.inventory_id;

revoke all on public.disposition_details from anon, authenticated;
grant select on public.disposition_details to service_role;

-- ---------------------------------------------------------------------
-- 6. Ringkasan notifikasi dihitung di database, bukan memuat ribuan barang.
-- ---------------------------------------------------------------------
create or replace function public.livira_dashboard_summary()
returns jsonb
language sql
stable
security definer
set search_path = public, pg_temp
as $$
with active_items as (
  select *
  from public.inventory_items
  where is_active = true
),
eligible_occupancy as (
  select distinct on (facility_id, unit_key)
    facility_id,
    case
      when upper(load_type) = 'FCL' then
        case upper(container_size)
          when '40' then 2::numeric
          when '40HC' then 2::numeric
          when '45' then 2.25::numeric
          when '45HC' then 2.25::numeric
          else 1::numeric
        end
      else 0::numeric
    end as yard_used,
    case
      when upper(load_type) = 'LCL' and estimated_volume_m3 > 0 then estimated_volume_m3
      else 0::numeric
    end as shed_used
  from (
    select ai.*,
      case
        when trim(ai.physical_unit_id) <> '' then trim(ai.physical_unit_id)
        when upper(ai.load_type) = 'FCL' and trim(ai.container_no) <> ''
          then 'FCL:' || upper(regexp_replace(ai.container_no, '[ .-]', '', 'g'))
        else 'ITEM:' || ai.id::text
      end as unit_key
    from active_items ai
    where ai.at_tpp = true
      and ai.facility_id is not null
      and (trim(ai.physical_unit_id) = '' or ai.occupancy_primary = true)
  ) candidates
  order by facility_id, unit_key, updated_at desc
),
facility_rows as (
  select
    f.id as facility_id,
    f.name as facility_name,
    count(ai.id) filter (where ai.item_type = 'BTD')::int as btd,
    count(ai.id) filter (where ai.item_type = 'BDN')::int as bdn,
    count(ai.id) filter (where ai.item_type = 'BMMN')::int as bmmn,
    count(ai.id)::int as total,
    f.yard_capacity,
    coalesce((select sum(eo.yard_used) from eligible_occupancy eo where eo.facility_id = f.id), 0) as yard_used,
    f.shed_capacity,
    coalesce((select sum(eo.shed_used) from eligible_occupancy eo where eo.facility_id = f.id), 0) as shed_used,
    f.sort_order
  from public.facilities f
  left join active_items ai on ai.facility_id = f.id
  where f.active = true
  group by f.id, f.name, f.yard_capacity, f.shed_capacity, f.sort_order
),
process_counts as (
  select
    count(*) filter (where is_active and disposition_type = 'lelang')::int as auction_active,
    count(*) filter (where is_active and disposition_type = 'musnah')::int as destruction_active,
    count(*) filter (where is_active and disposition_type = 'hibah')::int as grant_active,
    count(*) filter (
      where not is_active
        and date_trunc('month', updated_at) = date_trunc('month', now())
    )::int as completed_this_month
  from public.dispositions
),
attention as (
  select coalesce(jsonb_agg(to_jsonb(q) - 'search_text' order by q.determination_date asc), '[]'::jsonb) as rows
  from (
    select *
    from active_items
    where coalesce(determination_date, created_at) <= now() - interval '45 days'
    order by determination_date asc nulls first, created_at asc
    limit 5
  ) q
),
recent as (
  select coalesce(jsonb_agg(to_jsonb(q) order by q.created_at desc), '[]'::jsonb) as rows
  from (
    select *
    from public.events
    order by created_at desc
    limit 6
  ) q
),
item_counts as (
  select
    count(*)::int as active_total,
    count(*) filter (where item_type = 'BTD')::int as btd_total,
    count(*) filter (where item_type = 'BDN')::int as bdn_total,
    count(*) filter (where item_type = 'BMMN')::int as bmmn_total
  from active_items
),
facility_json as (
  select
    coalesce(jsonb_agg(
      jsonb_build_object(
        'facility_id', facility_id,
        'facility_name', facility_name,
        'btd', btd,
        'bdn', bdn,
        'bmmn', bmmn,
        'total', total,
        'yard_capacity', yard_capacity,
        'yard_used', yard_used,
        'shed_capacity', shed_capacity,
        'shed_used', shed_used
      ) order by sort_order, facility_name
    ), '[]'::jsonb) as rows,
    coalesce(sum(yard_capacity), 0) as yard_capacity,
    coalesce(sum(yard_used), 0) as yard_used,
    coalesce(sum(shed_capacity), 0) as shed_capacity,
    coalesce(sum(shed_used), 0) as shed_used
  from facility_rows
)
select jsonb_build_object(
  'active_total', ic.active_total,
  'btd_total', ic.btd_total,
  'bdn_total', ic.bdn_total,
  'bmmn_total', ic.bmmn_total,
  'auction_active', pc.auction_active,
  'destruction_active', pc.destruction_active,
  'grant_active', pc.grant_active,
  'completed_this_month', pc.completed_this_month,
  'occupancy', jsonb_build_object(
    'yard_capacity', fj.yard_capacity,
    'yard_used', fj.yard_used,
    'shed_capacity', fj.shed_capacity,
    'shed_used', fj.shed_used
  ),
  'facility_breakdown', fj.rows,
  'recent_events', r.rows,
  'attention_items', a.rows
)
from item_counts ic
cross join process_counts pc
cross join facility_json fj
cross join recent r
cross join attention a;
$$;

create or replace function public.livira_notification_summary(p_types text[] default null)
returns jsonb
language sql
stable
security definer
set search_path = public, pg_temp
as $$
  select jsonb_build_object(
    'overdue_60_days', count(*) filter (
      where item_type in ('BTD','BDN')
        and coalesce(determination_date, created_at) <= now() - interval '60 days'
        and status_code in ('masih_di_tps','ditetapkan')
    ),
    'ready_for_exit', count(*) filter (
      where status_code in ('laku','alokasi_hasil_lelang','ba_musnah','ba_serah_terima')
    ),
    'bmmn_waiting', count(*) filter (
      where item_type = 'BMMN' and current_disposition is null
    )
  )
  from public.inventory_items
  where is_active = true
    and (coalesce(cardinality(p_types), 0) = 0 or item_type = any(p_types));
$$;

-- Sumber data performa dibatasi di PostgreSQL hanya pada barang yang memiliki
-- event penyelesaian dalam periode terpilih. Perhitungan grouping tetap dilakukan
-- di Go agar kompatibel dengan aturan deduplikasi historis yang sudah diuji.
create or replace function public.livira_performance_source(
  p_from date,
  p_to date,
  p_types text[] default null
)
returns jsonb
language sql
stable
security definer
set search_path = public, pg_temp
as $$
with completion_ids as (
  select distinct e.inventory_id
  from public.events e
  join public.inventory_items i on i.id = e.inventory_id
  where i.item_type <> 'TITIPAN'
    and (coalesce(cardinality(p_types), 0) = 0 or i.item_type = any(p_types))
    and lower(e.code) = any(array[
      'selesai_lelang','laku','tidak_laku','ba_musnah','ba_serah_terima',
      'pencacahan','penelitian_pfpd','penelitian_hs_lartas','penetapan_bmmn'
    ]::text[])
    and coalesce(e.document_date, e.created_at) >= p_from::timestamptz
    and coalesce(e.document_date, e.created_at) < (p_to + 1)::timestamptz
),
item_rows as (
  select coalesce(
    jsonb_agg(to_jsonb(i) - 'search_text' order by i.id),
    '[]'::jsonb
  ) as rows
  from public.inventory_items i
  join completion_ids c on c.inventory_id = i.id
),
event_rows as (
  select coalesce(
    jsonb_agg(to_jsonb(e) order by e.created_at, e.id),
    '[]'::jsonb
  ) as rows
  from public.events e
  join completion_ids c on c.inventory_id = e.inventory_id
  where lower(e.code) = any(array[
    'ditetapkan','masih_di_tps','request_penelitian_pfpd','siap_peruntukan',
    'selesai_lelang','laku','tidak_laku','ba_musnah','ba_serah_terima',
    'pencacahan','penelitian_pfpd','penelitian_hs_lartas','penetapan_bmmn'
  ]::text[])
)
select jsonb_build_object('items', i.rows, 'events', e.rows)
from item_rows i
cross join event_rows e;
$$;

-- Agregasi laporan inventory dihitung di database; halaman pelaporan hanya
-- mengambil satu halaman data, sementara kartu ringkasan tetap mencakup semua hasil.
create or replace function public.livira_inventory_summary(
  p_query text default '',
  p_types text[] default null,
  p_facility_id text default '',
  p_item_type text default '',
  p_status text default '',
  p_item_kind text default '',
  p_goods_condition text default '',
  p_category text default '',
  p_allocation_purpose text default '',
  p_location_scope text default '',
  p_include_inactive boolean default false,
  p_only_inactive boolean default false,
  p_date_from date default null,
  p_date_to date default null,
  p_age_before date default null,
  p_min_value bigint default 0,
  p_max_value bigint default 0,
  p_preset text default ''
)
returns jsonb
language sql
stable
security definer
set search_path = public, pg_temp
as $$
with filtered as (
  select i.*
  from public.inventory_items i
  where
    case
      when p_only_inactive then not i.is_active
      when p_include_inactive then true
      else i.is_active
    end
    and (coalesce(cardinality(p_types), 0) = 0 or i.item_type = any(p_types))
    and (coalesce(p_item_type, '') = '' or i.item_type = p_item_type)
    and (coalesce(p_facility_id, '') = '' or i.facility_id = p_facility_id)
    and (coalesce(p_status, '') = '' or i.status_code = p_status)
    and (coalesce(p_item_kind, '') = '' or i.item_kind = p_item_kind)
    and (coalesce(p_goods_condition, '') = '' or i.goods_condition = p_goods_condition)
    and (coalesce(p_category, '') = '' or i.category = p_category)
    and (coalesce(p_allocation_purpose, '') = '' or lower(i.allocation_purpose) = lower(p_allocation_purpose))
    and (
      coalesce(p_location_scope, '') = ''
      or (p_location_scope = 'tpp' and i.at_tpp)
      or (p_location_scope = 'tps' and not i.at_tpp)
    )
    and (p_date_from is null or i.determination_date >= p_date_from::timestamptz)
    and (p_date_to is null or i.determination_date < (p_date_to + 1)::timestamptz)
    and (p_age_before is null or i.determination_date < (p_age_before + 1)::timestamptz)
    and (coalesce(p_min_value, 0) <= 0 or i.goods_value >= p_min_value)
    and (coalesce(p_max_value, 0) <= 0 or i.goods_value <= p_max_value)
    and (coalesce(p_query, '') = '' or i.search_text ilike '%' || lower(p_query) || '%')
    and case coalesce(p_preset, '')
      when 'overdue_60' then
        i.item_type in ('BTD', 'BDN')
        and i.status_code in ('masih_di_tps', 'ditetapkan')
      when 'auction_ready' then
        i.goods_value > 0
        and i.current_disposition is null
        and (i.status_code = 'penelitian_pfpd' or i.item_type = 'BMMN')
      when 'bmmn_allocation' then
        i.item_type = 'BMMN' and i.current_disposition is null
      else true
    end
)
select jsonb_build_object(
  'total', count(*)::int,
  'total_value', coalesce(sum(goods_value), 0)::bigint,
  'at_tpp', count(*) filter (where at_tpp)::int,
  'active', count(*) filter (where is_active)::int,
  'closed', count(*) filter (where not is_active)::int
)
from filtered;
$$;

-- Ringkasan dashboard tiap menu proses dihitung di PostgreSQL agar halaman
-- tidak perlu memuat seluruh riwayat lelang/musnah/hibah ke memori aplikasi.
create or replace function public.livira_process_dashboard(
  p_type text,
  p_year integer,
  p_types text[] default null
)
returns jsonb
language sql
stable
security definer
set search_path = public, pg_temp
as $$
with filtered as (
  select *
  from public.disposition_details
  where disposition_type = p_type
    and (coalesce(cardinality(p_types), 0) = 0 or inventory_item_type = any(p_types))
),
months as (
  select generate_series(1, 12)::int as month_no
),
monthly as (
  select
    m.month_no,
    count(d.id)::int as count,
    coalesce(sum(d.inventory_goods_value) filter (where p_type = 'lelang'), 0)::bigint as goods_value,
    coalesce(sum(d.htl_value) filter (where p_type = 'lelang'), 0)::bigint as htl_value,
    coalesce(sum(d.sale_value) filter (where p_type = 'lelang'), 0)::bigint as sale_value,
    coalesce(sum(d.destruction_cost) filter (where p_type = 'musnah'), 0)::bigint as cost,
    count(d.id) filter (where p_type = 'hibah' and d.transfer_type = 'hibah')::int as grant_count,
    count(d.id) filter (where p_type = 'hibah' and d.transfer_type = 'psp')::int as psp
  from months m
  left join filtered d
    on extract(year from d.created_at)::int = p_year
   and extract(month from d.created_at)::int = m.month_no
  group by m.month_no
),
stats as (
  select
    count(*) filter (where is_active)::int as active,
    count(*) filter (where extract(year from created_at)::int = p_year)::int as started_this_year,
    count(*) filter (
      where extract(year from updated_at)::int = p_year
        and (
          not is_active
          or (
            p_type = 'lelang'
            and status_code in ('laku', 'tidak_laku', 'alokasi_hasil_lelang')
          )
        )
    )::int as completed_this_year,
    coalesce(sum(inventory_goods_value) filter (
      where p_type = 'lelang' and extract(year from created_at)::int = p_year
    ), 0)::bigint as total_goods_value,
    coalesce(sum(htl_value) filter (
      where p_type = 'lelang' and extract(year from created_at)::int = p_year
    ), 0)::bigint as total_htl_value,
    coalesce(sum(sale_value) filter (
      where p_type = 'lelang' and extract(year from created_at)::int = p_year
    ), 0)::bigint as total_sale_value,
    coalesce(sum(destruction_cost) filter (
      where p_type = 'musnah' and extract(year from created_at)::int = p_year
    ), 0)::bigint as total_cost,
    count(*) filter (
      where p_type = 'hibah'
        and transfer_type = 'hibah'
        and extract(year from created_at)::int = p_year
    )::int as total_grant,
    count(*) filter (
      where p_type = 'hibah'
        and transfer_type = 'psp'
        and extract(year from created_at)::int = p_year
    )::int as total_psp
  from filtered
),
chart as (
  select coalesce(jsonb_agg(
    jsonb_build_object(
      'label', case month_no
        when 1 then 'Jan' when 2 then 'Feb' when 3 then 'Mar'
        when 4 then 'Apr' when 5 then 'Mei' when 6 then 'Jun'
        when 7 then 'Jul' when 8 then 'Agu' when 9 then 'Sep'
        when 10 then 'Okt' when 11 then 'Nov' else 'Des'
      end,
      'count', count,
      'goods_value', goods_value,
      'htl_value', htl_value,
      'sale_value', sale_value,
      'cost', cost,
      'grant', grant_count,
      'psp', psp
    ) order by month_no
  ), '[]'::jsonb) as rows
  from monthly
),
maximums as (
  select
    greatest(1, coalesce(max(greatest(count, grant_count, psp)), 0))::int as max_count,
    greatest(1::bigint, coalesce(max(greatest(goods_value, htl_value, sale_value, cost)), 0))::bigint as max_money
  from monthly
)
select jsonb_build_object(
  'year', p_year,
  'active', s.active,
  'this_year', s.started_this_year,
  'started_this_year', s.started_this_year,
  'completed_this_year', s.completed_this_year,
  'total_goods_value', s.total_goods_value,
  'total_htl_value', s.total_htl_value,
  'total_sale_value', s.total_sale_value,
  'total_cost', s.total_cost,
  'total_grant', s.total_grant,
  'total_psp', s.total_psp,
  'max_count', mx.max_count,
  'max_money', mx.max_money,
  'chart', c.rows
)
from stats s
cross join chart c
cross join maximums mx;
$$;

-- ---------------------------------------------------------------------
-- 7. Workflow atomik dan optimistic locking.
-- ---------------------------------------------------------------------
create or replace function public.livira_create_inventories(p_rows jsonb)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_entry jsonb;
  v_item public.inventory_items%rowtype;
  v_process_id uuid;
  v_initial_type text;
  v_active boolean;
  v_result jsonb := '[]'::jsonb;
begin
  if jsonb_typeof(p_rows) <> 'array'
     or jsonb_array_length(p_rows) = 0
     or jsonb_array_length(p_rows) > 500 then
    raise exception 'invalid transition: inventory batch size' using errcode = '22023';
  end if;

  for v_entry in select value from jsonb_array_elements(p_rows)
  loop
    insert into public.inventory_items (
      reference_no, item_type, origin_type,
      manifest_no, manifest_date, manifest_position,
      determination_no, determination_date, category,
      entrusted_category, source_office,
      description, item_kind, quantity, unit, goods_value, goods_condition,
      location, location_status, at_tpp, owner_name, owner_address,
      origin_warehouse, facility_id, facility_name,
      load_type, container_no, container_size, estimated_volume_m3,
      physical_unit_id, occupancy_primary, pfpd_required,
      restriction_rule, status_code, status_label, current_disposition,
      is_active, created_by
    ) values (
      coalesce(v_entry->>'reference_no', ''),
      coalesce(v_entry->>'item_type', ''),
      coalesce(v_entry->>'origin_type', ''),
      coalesce(v_entry->>'manifest_no', ''),
      nullif(v_entry->>'manifest_date', '')::timestamptz,
      coalesce(v_entry->>'manifest_position', ''),
      coalesce(v_entry->>'determination_no', ''),
      nullif(v_entry->>'determination_date', '')::timestamptz,
      coalesce(v_entry->>'category', ''),
      coalesce(v_entry->>'entrusted_category', ''),
      coalesce(v_entry->>'source_office', ''),
      coalesce(v_entry->>'description', ''),
      coalesce(v_entry->>'item_kind', ''),
      coalesce(nullif(v_entry->>'quantity', '')::numeric, 0),
      coalesce(v_entry->>'unit', ''),
      coalesce(nullif(v_entry->>'goods_value', '')::bigint, 0),
      coalesce(v_entry->>'goods_condition', ''),
      coalesce(v_entry->>'location', ''),
      coalesce(v_entry->>'location_status', ''),
      coalesce(nullif(v_entry->>'at_tpp', '')::boolean, false),
      coalesce(v_entry->>'owner_name', ''),
      coalesce(v_entry->>'owner_address', ''),
      coalesce(v_entry->>'origin_warehouse', ''),
      nullif(v_entry->>'facility_id', ''),
      coalesce(v_entry->>'facility_name', ''),
      coalesce(v_entry->>'load_type', ''),
      coalesce(v_entry->>'container_no', ''),
      coalesce(v_entry->>'container_size', ''),
      coalesce(nullif(v_entry->>'estimated_volume_m3', '')::numeric, 0),
      coalesce(v_entry->>'physical_unit_id', ''),
      coalesce(nullif(v_entry->>'occupancy_primary', '')::boolean, true),
      coalesce(nullif(v_entry->>'pfpd_required', '')::boolean, true),
      coalesce(v_entry->>'restriction_rule', ''),
      coalesce(v_entry->>'status_code', ''),
      coalesce(v_entry->>'status_label', ''),
      null,
      coalesce(nullif(v_entry->>'is_active', '')::boolean, true),
      coalesce(v_entry->>'created_by', '')
    ) returning * into v_item;

    v_process_id := null;
    v_initial_type := nullif(trim(coalesce(v_entry->>'_initial_disposition_type', '')), '');
    if v_initial_type is not null then
      if v_initial_type not in ('lelang', 'musnah', 'hibah') then
        raise exception 'invalid disposition type' using errcode = '22023';
      end if;
      v_active := coalesce(v_entry->>'_initial_status_code', '') not in (
        'alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima'
      );
      insert into public.dispositions (
        inventory_id, disposition_type, round, status_code, status_label,
        schedule_document_no, schedule_document_date, transfer_type,
        is_active, created_by
      ) values (
        v_item.id, v_initial_type, 1, v_item.status_code, v_item.status_label,
        case when coalesce(v_entry->>'_initial_status_code', '') = 'jadwal_lelang' then v_item.determination_no else '' end,
        case when coalesce(v_entry->>'_initial_status_code', '') = 'jadwal_lelang' then v_item.determination_date else null end,
        coalesce(v_entry->>'_initial_transfer_type', ''),
        v_active, v_item.created_by
      ) returning id into v_process_id;

      if v_active then
        update public.inventory_items
        set current_disposition = v_initial_type
        where id = v_item.id
        returning * into v_item;
      end if;
    end if;

    insert into public.events (
      inventory_id, disposition_id, disposition_type, code, label,
      document_no, document_date, notes, actor, document_id
    ) values (
      v_item.id, v_process_id, v_initial_type, v_item.status_code, v_item.status_label,
      v_item.determination_no, v_item.determination_date,
      case when coalesce(nullif(v_entry->>'_reconciliation_created', ''), 'false')::boolean
        then 'Inventory ditambahkan melalui rekonsiliasi kondisi fisik.' else '' end,
      v_item.created_by,
      nullif(v_entry->>'_document_id', '')::uuid
    );

    v_result := v_result || jsonb_build_array(to_jsonb(v_item) - 'search_text');
  end loop;

  return v_result;
end;
$$;

create or replace function public.livira_create_disposition(
  p_inventory_id uuid,
  p_disposition_type text,
  p_actor text,
  p_notes text default '',
  p_expected_updated_at timestamptz default null
)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_item public.inventory_items%rowtype;
  v_process public.dispositions%rowtype;
  v_code text;
  v_label text;
  v_result jsonb;
begin
  select * into v_item
  from public.inventory_items
  where id = p_inventory_id
  for update;

  if not found then
    raise exception 'not found: inventory' using errcode = 'P0002';
  end if;
  if not v_item.is_active then
    raise exception 'inventory is inactive' using errcode = 'P0001';
  end if;
  if p_expected_updated_at is not null and v_item.updated_at is distinct from p_expected_updated_at then
    raise exception 'record changed by another user' using errcode = '40001';
  end if;
  if v_item.current_disposition is not null
     or exists (select 1 from public.dispositions where inventory_id = v_item.id and is_active) then
    raise exception 'active disposition already exists' using errcode = '23505';
  end if;

  case p_disposition_type
    when 'lelang' then v_label := 'Proses lelang dimulai';
    when 'musnah' then v_label := 'Proses pemusnahan dimulai';
    when 'hibah' then v_label := 'Proses hibah/PSP dimulai';
    else raise exception 'invalid disposition type' using errcode = '22023';
  end case;
  v_code := 'proses_' || p_disposition_type;

  insert into public.dispositions (
    inventory_id, disposition_type, round, status_code, status_label,
    is_active, created_by
  ) values (
    v_item.id, p_disposition_type, 1, v_code, v_label,
    true, coalesce(p_actor, '')
  ) returning * into v_process;

  update public.inventory_items
  set current_disposition = p_disposition_type,
      status_code = v_code,
      status_label = v_label
  where id = v_item.id;

  insert into public.events (
    inventory_id, disposition_id, disposition_type, code, label, notes, actor
  ) values (
    v_item.id, v_process.id, p_disposition_type, v_code, v_label,
    coalesce(p_notes, ''), coalesce(p_actor, '')
  );

  select to_jsonb(v) into v_result
  from public.disposition_details v
  where v.id = v_process.id;
  return v_result;
end;
$$;

create or replace function public.livira_apply_disposition_event(
  p_disposition_id uuid,
  p_expected_updated_at timestamptz,
  p_process_patch jsonb,
  p_item_patch jsonb,
  p_event jsonb
)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_process public.dispositions%rowtype;
  v_new_process public.dispositions%rowtype;
  v_item public.inventory_items%rowtype;
  v_new_item public.inventory_items%rowtype;
  v_result jsonb;
begin
  select * into v_process
  from public.dispositions
  where id = p_disposition_id
  for update;
  if not found then
    raise exception 'not found: disposition' using errcode = 'P0002';
  end if;
  if not v_process.is_active then
    raise exception 'invalid transition: disposition inactive' using errcode = 'P0001';
  end if;
  if p_expected_updated_at is not null and v_process.updated_at is distinct from p_expected_updated_at then
    raise exception 'record changed by another user' using errcode = '40001';
  end if;

  select * into v_item
  from public.inventory_items
  where id = v_process.inventory_id
  for update;
  if not found then
    raise exception 'not found: inventory' using errcode = 'P0002';
  end if;

  select * into v_new_process
  from jsonb_populate_record(v_process, coalesce(p_process_patch, '{}'::jsonb));
  select * into v_new_item
  from jsonb_populate_record(v_item, coalesce(p_item_patch, '{}'::jsonb));

  update public.dispositions
  set round = v_new_process.round,
      status_code = v_new_process.status_code,
      status_label = v_new_process.status_label,
      proposal_type = v_new_process.proposal_type,
      recipient_code = v_new_process.recipient_code,
      recipient_name = v_new_process.recipient_name,
      sale_value = v_new_process.sale_value,
      htl_value = v_new_process.htl_value,
      execution_start_date = v_new_process.execution_start_date,
      execution_end_date = v_new_process.execution_end_date,
      schedule_document_no = v_new_process.schedule_document_no,
      schedule_document_date = v_new_process.schedule_document_date,
      auction_outcome = v_new_process.auction_outcome,
      allocation_target = v_new_process.allocation_target,
      destruction_cost = v_new_process.destruction_cost,
      transfer_type = v_new_process.transfer_type,
      is_active = v_new_process.is_active
  where id = v_process.id;

  update public.inventory_items
  set status_code = v_new_item.status_code,
      status_label = v_new_item.status_label,
      current_disposition = v_new_item.current_disposition
  where id = v_item.id;

  insert into public.events (
    inventory_id, disposition_id, disposition_type, code, label,
    document_no, document_date, notes, actor, document_id
  ) values (
    v_item.id, v_process.id, v_process.disposition_type,
    coalesce(p_event->>'code', ''), coalesce(p_event->>'label', ''),
    coalesce(p_event->>'document_no', ''), nullif(p_event->>'document_date', '')::timestamptz,
    coalesce(p_event->>'notes', ''), coalesce(p_event->>'actor', ''),
    nullif(p_event->>'document_id', '')::uuid
  );

  select to_jsonb(v) into v_result
  from public.disposition_details v
  where v.id = v_process.id;
  return v_result;
end;
$$;

create or replace function public.livira_apply_inventory_event(
  p_inventory_id uuid,
  p_expected_updated_at timestamptz,
  p_item_patch jsonb,
  p_close_active_dispositions boolean default false,
  p_keep_destruction_open boolean default false,
  p_event jsonb default '{}'::jsonb
)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_item public.inventory_items%rowtype;
  v_new_item public.inventory_items%rowtype;
  v_result jsonb;
begin
  select * into v_item
  from public.inventory_items
  where id = p_inventory_id
  for update;
  if not found then
    raise exception 'not found: inventory' using errcode = 'P0002';
  end if;
  if not v_item.is_active then
    raise exception 'inventory is inactive' using errcode = 'P0001';
  end if;
  if p_expected_updated_at is not null and v_item.updated_at is distinct from p_expected_updated_at then
    raise exception 'record changed by another user' using errcode = '40001';
  end if;

  select * into v_new_item
  from jsonb_populate_record(v_item, coalesce(p_item_patch, '{}'::jsonb));

  update public.inventory_items
  set item_type = v_new_item.item_type,
      determination_no = v_new_item.determination_no,
      determination_date = v_new_item.determination_date,
      description = v_new_item.description,
      item_kind = v_new_item.item_kind,
      quantity = v_new_item.quantity,
      unit = v_new_item.unit,
      goods_value = v_new_item.goods_value,
      goods_condition = v_new_item.goods_condition,
      location = v_new_item.location,
      location_status = v_new_item.location_status,
      at_tpp = v_new_item.at_tpp,
      facility_id = v_new_item.facility_id,
      facility_name = v_new_item.facility_name,
      pfpd_required = v_new_item.pfpd_required,
      research_request_no = v_new_item.research_request_no,
      research_request_date = v_new_item.research_request_date,
      hs_code = v_new_item.hs_code,
      is_restricted = v_new_item.is_restricted,
      restriction_rule = v_new_item.restriction_rule,
      origin_document_type = v_new_item.origin_document_type,
      origin_document_no = v_new_item.origin_document_no,
      origin_document_date = v_new_item.origin_document_date,
      allocation_purpose = v_new_item.allocation_purpose,
      allocation_proposal_type = v_new_item.allocation_proposal_type,
      allocation_proposal_no = v_new_item.allocation_proposal_no,
      allocation_proposal_date = v_new_item.allocation_proposal_date,
      allocation_approval_type = v_new_item.allocation_approval_type,
      allocation_approval_no = v_new_item.allocation_approval_no,
      allocation_approval_date = v_new_item.allocation_approval_date,
      exit_document_no = v_new_item.exit_document_no,
      exit_document_date = v_new_item.exit_document_date,
      exit_type = v_new_item.exit_type,
      exit_notes = v_new_item.exit_notes,
      status_code = v_new_item.status_code,
      status_label = v_new_item.status_label,
      current_disposition = v_new_item.current_disposition,
      is_active = v_new_item.is_active
  where id = v_item.id;

  if coalesce(p_close_active_dispositions, false) then
    update public.dispositions
    set is_active = false
    where inventory_id = v_item.id
      and is_active = true
      and (
        not coalesce(p_keep_destruction_open, false)
        or disposition_type <> 'musnah'
      );
  end if;

  insert into public.events (
    inventory_id, code, label, document_no, document_date,
    notes, actor, document_id
  ) values (
    v_item.id, coalesce(p_event->>'code', ''), coalesce(p_event->>'label', ''),
    coalesce(p_event->>'document_no', ''), nullif(p_event->>'document_date', '')::timestamptz,
    coalesce(p_event->>'notes', ''), coalesce(p_event->>'actor', ''),
    nullif(p_event->>'document_id', '')::uuid
  );

  select to_jsonb(i) - 'search_text' into v_result
  from public.inventory_items i
  where i.id = v_item.id;
  return v_result;
end;
$$;

-- ---------------------------------------------------------------------
-- 8. Edit label TPP mempertahankan kode dan menyinkronkan nama tampilan.
-- ---------------------------------------------------------------------
create or replace function public.livira_update_facility_parameter(
  p_facility_id text,
  p_name text,
  p_sort_order integer default 999
)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_old public.facilities%rowtype;
  v_new public.facilities%rowtype;
begin
  select * into v_old
  from public.facilities
  where id = p_facility_id
  for update;
  if not found then
    raise exception 'not found: facility' using errcode = 'P0002';
  end if;
  if coalesce(trim(p_name), '') = '' then
    raise exception 'invalid transition: facility name is empty' using errcode = '22023';
  end if;

  update public.facilities
  set name = trim(p_name),
      sort_order = case when p_sort_order > 0 then p_sort_order else sort_order end
  where id = p_facility_id
  returning * into v_new;

  update public.inventory_items
  set facility_name = v_new.name,
      location = case
        when at_tpp and (trim(location) = '' or location = v_old.name) then v_new.name
        else location
      end,
      location_status = case
        when at_tpp and (trim(location_status) = '' or location_status = v_old.name) then v_new.name
        else location_status
      end
  where facility_id = p_facility_id;

  return to_jsonb(v_new);
end;
$$;

-- ---------------------------------------------------------------------
-- 9. Indeks tambahan untuk workflow/timeline dan agregasi performa.
-- ---------------------------------------------------------------------
create index if not exists dispositions_inventory_active_updated_idx
  on public.dispositions (inventory_id, is_active, updated_at desc);
create index if not exists dispositions_type_active_updated_idx
  on public.dispositions (disposition_type, is_active, updated_at desc);
create index if not exists events_document_inventory_idx
  on public.events (document_id, inventory_id)
  where document_id is not null;
create index if not exists events_inventory_code_date_idx
  on public.events (inventory_id, code, document_date, created_at);

-- Semua RPC hanya boleh dipanggil backend service role.
revoke all on function public.livira_dashboard_summary() from public, anon, authenticated;
revoke all on function public.livira_notification_summary(text[]) from public, anon, authenticated;
revoke all on function public.livira_performance_source(date,date,text[]) from public, anon, authenticated;
revoke all on function public.livira_inventory_summary(text,text[],text,text,text,text,text,text,text,text,boolean,boolean,date,date,date,bigint,bigint,text) from public, anon, authenticated;
revoke all on function public.livira_process_dashboard(text,integer,text[]) from public, anon, authenticated;
revoke all on function public.livira_create_inventories(jsonb) from public, anon, authenticated;
revoke all on function public.livira_create_disposition(uuid,text,text,text,timestamptz) from public, anon, authenticated;
revoke all on function public.livira_apply_disposition_event(uuid,timestamptz,jsonb,jsonb,jsonb) from public, anon, authenticated;
revoke all on function public.livira_apply_inventory_event(uuid,timestamptz,jsonb,boolean,boolean,jsonb) from public, anon, authenticated;
revoke all on function public.livira_update_facility_parameter(text,text,integer) from public, anon, authenticated;

grant execute on function public.livira_dashboard_summary() to service_role;
grant execute on function public.livira_notification_summary(text[]) to service_role;
grant execute on function public.livira_performance_source(date,date,text[]) to service_role;
grant execute on function public.livira_inventory_summary(text,text[],text,text,text,text,text,text,text,text,boolean,boolean,date,date,date,bigint,bigint,text) to service_role;
grant execute on function public.livira_process_dashboard(text,integer,text[]) to service_role;
grant execute on function public.livira_create_inventories(jsonb) to service_role;
grant execute on function public.livira_create_disposition(uuid,text,text,text,timestamptz) to service_role;
grant execute on function public.livira_apply_disposition_event(uuid,timestamptz,jsonb,jsonb,jsonb) to service_role;
grant execute on function public.livira_apply_inventory_event(uuid,timestamptz,jsonb,boolean,boolean,jsonb) to service_role;
grant execute on function public.livira_update_facility_parameter(text,text,integer) to service_role;

analyze public.inventory_items;
analyze public.dispositions;
analyze public.events;

-- Meminta PostgREST memuat view/RPC baru segera setelah transaksi commit.
notify pgrst, 'reload schema';

commit;

-- END MIGRATION: 016_security_performance_hardening.sql

-- ============================================================================
-- PATCH KONSISTENSI KHUSUS FRESH INSTALL
-- ============================================================================
-- 001_schema.sql versi terbaru sudah mempunyai CHECK ukuran kontainer lama.
-- Migration 011 menambahkan 40HC/45HC dengan nama constraint baru. Pada fresh
-- install, constraint lama harus dibuang agar 40HC dan 45HC dapat digunakan.

begin;

alter table public.inventory_items
  drop constraint if exists inventory_items_container_size_check,
  drop constraint if exists inventory_container_size_check,
  drop constraint if exists inventory_fcl_container_detail_check;

alter table public.inventory_items
  add constraint inventory_container_size_check
    check (upper(container_size) in ('', '20', '40', '40HC', '45HC')),
  add constraint inventory_fcl_container_detail_check
    check (
      upper(load_type) <> 'FCL'
      or container_no = ''
      or upper(container_size) in ('20', '40', '40HC', '45HC')
    );

-- Master TPP saja; tidak ada barang dummy.
insert into public.facilities (
  id, name, active, sort_order,
  yard_capacity, yard_used, shed_capacity, shed_used
) values
  ('tpp-transporindo',       'TPP Transporindo',       true, 1, 0, 0, 0, 0),
  ('tpp-multi-sejahtera',    'TPP Multi Sejahtera',    true, 2, 0, 0, 0, 0),
  ('tpp-kbn-marunda',        'TPP KBN Marunda',        true, 3, 0, 0, 0, 0),
  ('tpp-graha-segara',       'TPP Graha Segara',       true, 4, 0, 0, 0, 0)
on conflict (id) do update set
  name = excluded.name,
  active = excluded.active,
  sort_order = excluded.sort_order;

commit;


-- ============================================================================
-- BEGIN MIGRATION: 017_transfer_lelang_rekonsiliasi_perubahan_data.sql
-- ============================================================================
-- LIVIRA migration 017
-- 1. Barang lelang berstatus Tidak Laku dapat dialihkan secara atomik ke
--    pemusnahan atau hibah/PSP.
-- 2. Rekonsiliasi memperoleh jenis ketiga: Perubahan data barang.
-- 3. Seluruh data bisnis, dokumen timeline, dan data nilai proses dapat
--    dikoreksi tanpa mengubah ID sistem maupun konsistensi status alur.

begin;

alter table public.reconciliations
  drop constraint if exists reconciliations_reconciliation_type_check;
alter table public.reconciliations
  add constraint reconciliations_reconciliation_type_check
  check (reconciliation_type in ('recorded_not_found','found_not_recorded','data_correction'));

alter table public.reconciliations
  drop constraint if exists reconciliations_action_check;
alter table public.reconciliations
  add constraint reconciliations_action_check
  check (action in ('removed','added','updated'));

create or replace function public.livira_create_disposition(
  p_inventory_id uuid,
  p_disposition_type text,
  p_actor text,
  p_notes text default '',
  p_expected_updated_at timestamptz default null
)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_item public.inventory_items%rowtype;
  v_process public.dispositions%rowtype;
  v_old_process public.dispositions%rowtype;
  v_code text;
  v_label text;
  v_transfer_code text;
  v_transfer_label text;
  v_transfer_failed_auction boolean := false;
  v_result jsonb;
begin
  select * into v_item
  from public.inventory_items
  where id = p_inventory_id
  for update;

  if not found then
    raise exception 'not found: inventory' using errcode = 'P0002';
  end if;
  if not v_item.is_active then
    raise exception 'inventory is inactive' using errcode = 'P0001';
  end if;
  if p_expected_updated_at is not null and v_item.updated_at is distinct from p_expected_updated_at then
    raise exception 'record changed by another user' using errcode = '40001';
  end if;
  if p_disposition_type not in ('lelang','musnah','hibah') then
    raise exception 'invalid disposition type' using errcode = '22023';
  end if;

  v_transfer_failed_auction :=
    v_item.current_disposition = 'lelang'
    and v_item.status_code = 'tidak_laku'
    and p_disposition_type in ('musnah','hibah');

  if v_item.current_disposition is not null and not v_transfer_failed_auction then
    raise exception 'active disposition already exists' using errcode = '23505';
  end if;

  if v_transfer_failed_auction then
    select * into v_old_process
    from public.dispositions
    where inventory_id = v_item.id
      and disposition_type = 'lelang'
      and is_active = true
      and status_code = 'tidak_laku'
    order by updated_at desc
    limit 1
    for update;

    if not found then
      raise exception 'active failed auction disposition not found' using errcode = '23505';
    end if;

    if p_disposition_type = 'musnah' then
      v_transfer_code := 'dialihkan_musnah';
      v_transfer_label := 'Dialihkan ke pemusnahan';
    else
      v_transfer_code := 'dialihkan_hibah';
      v_transfer_label := 'Dialihkan ke hibah/PSP';
    end if;

    update public.dispositions
    set status_code = v_transfer_code,
        status_label = v_transfer_label,
        is_active = false
    where id = v_old_process.id;

    insert into public.events (
      inventory_id, disposition_id, disposition_type, code, label, notes, actor
    ) values (
      v_item.id, v_old_process.id, 'lelang', v_transfer_code, v_transfer_label,
      'Barang lelang tidak laku dialihkan ke proses ' ||
        case when p_disposition_type = 'musnah' then 'pemusnahan.' else 'hibah/PSP.' end,
      coalesce(p_actor, '')
    );
  elsif exists (
    select 1 from public.dispositions
    where inventory_id = v_item.id and is_active
  ) then
    raise exception 'active disposition already exists' using errcode = '23505';
  end if;

  case p_disposition_type
    when 'lelang' then v_label := 'Proses lelang dimulai';
    when 'musnah' then v_label := 'Proses pemusnahan dimulai';
    when 'hibah' then v_label := 'Proses hibah/PSP dimulai';
  end case;
  v_code := 'proses_' || p_disposition_type;

  insert into public.dispositions (
    inventory_id, disposition_type, round, status_code, status_label,
    is_active, created_by
  ) values (
    v_item.id, p_disposition_type, 1, v_code, v_label,
    true, coalesce(p_actor, '')
  ) returning * into v_process;

  update public.inventory_items
  set current_disposition = p_disposition_type,
      status_code = v_code,
      status_label = v_label
  where id = v_item.id;

  insert into public.events (
    inventory_id, disposition_id, disposition_type, code, label, notes, actor
  ) values (
    v_item.id, v_process.id, p_disposition_type, v_code, v_label,
    coalesce(p_notes, ''), coalesce(p_actor, '')
  );

  select to_jsonb(v) into v_result
  from public.disposition_details v
  where v.id = v_process.id;
  return v_result;
end;
$$;

create or replace function public.livira_correct_inventory_data(
  p_inventory_id uuid,
  p_actor text,
  p_reason text,
  p_item_patch jsonb,
  p_event_patches jsonb default '[]'::jsonb,
  p_process_patches jsonb default '[]'::jsonb,
  p_document_id uuid default null,
  p_expected_updated_at timestamptz default null
)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_item public.inventory_items%rowtype;
  v_new_item public.inventory_items%rowtype;
  v_event public.events%rowtype;
  v_new_event public.events%rowtype;
  v_process public.dispositions%rowtype;
  v_new_process public.dispositions%rowtype;
  v_record public.reconciliations%rowtype;
  v_patch jsonb;
  v_notes text;
begin
  if coalesce(btrim(p_actor), '') = '' then
    raise exception 'invalid actor' using errcode = '22023';
  end if;
  if p_reason not in ('Kesalahan input', 'Error pada saat pengisian awal') then
    raise exception 'invalid correction reason' using errcode = '22023';
  end if;
  if jsonb_typeof(coalesce(p_item_patch, '{}'::jsonb)) <> 'object'
     or jsonb_typeof(coalesce(p_event_patches, '[]'::jsonb)) <> 'array'
     or jsonb_typeof(coalesce(p_process_patches, '[]'::jsonb)) <> 'array' then
    raise exception 'invalid correction payload' using errcode = '22023';
  end if;

  select * into v_item
  from public.inventory_items
  where id = p_inventory_id
  for update;
  if not found then
    raise exception 'not found: inventory' using errcode = 'P0002';
  end if;
  if p_expected_updated_at is not null and v_item.updated_at is distinct from p_expected_updated_at then
    raise exception 'record changed by another user' using errcode = '40001';
  end if;

  select * into v_new_item
  from jsonb_populate_record(v_item, coalesce(p_item_patch, '{}'::jsonb));

  if coalesce(btrim(v_new_item.reference_no), '') = ''
     or coalesce(btrim(v_new_item.determination_no), '') = ''
     or v_new_item.determination_date is null
     or coalesce(btrim(v_new_item.description), '') = '' then
    raise exception 'required inventory data is empty' using errcode = '22023';
  end if;

  if v_new_item.facility_id is not null then
    select name into v_new_item.facility_name
    from public.facilities
    where id = v_new_item.facility_id and active = true;
    if not found then
      raise exception 'invalid facility' using errcode = '22023';
    end if;
  else
    v_new_item.facility_name := '';
  end if;

  update public.inventory_items
  set reference_no = v_new_item.reference_no,
      item_type = v_new_item.item_type,
      origin_type = v_new_item.origin_type,
      manifest_no = v_new_item.manifest_no,
      manifest_date = v_new_item.manifest_date,
      manifest_position = v_new_item.manifest_position,
      determination_no = v_new_item.determination_no,
      determination_date = v_new_item.determination_date,
      category = v_new_item.category,
      entrusted_category = v_new_item.entrusted_category,
      source_office = v_new_item.source_office,
      description = v_new_item.description,
      item_kind = v_new_item.item_kind,
      quantity = v_new_item.quantity,
      unit = v_new_item.unit,
      goods_value = v_new_item.goods_value,
      goods_condition = v_new_item.goods_condition,
      location = v_new_item.location,
      location_status = v_new_item.location_status,
      at_tpp = v_new_item.at_tpp,
      owner_name = v_new_item.owner_name,
      owner_address = v_new_item.owner_address,
      origin_warehouse = v_new_item.origin_warehouse,
      facility_id = v_new_item.facility_id,
      facility_name = v_new_item.facility_name,
      load_type = v_new_item.load_type,
      container_no = v_new_item.container_no,
      container_size = v_new_item.container_size,
      estimated_volume_m3 = v_new_item.estimated_volume_m3,
      physical_unit_id = v_new_item.physical_unit_id,
      occupancy_primary = v_new_item.occupancy_primary,
      pfpd_required = v_new_item.pfpd_required,
      research_request_no = v_new_item.research_request_no,
      research_request_date = v_new_item.research_request_date,
      hs_code = v_new_item.hs_code,
      is_restricted = v_new_item.is_restricted,
      restriction_rule = v_new_item.restriction_rule,
      origin_document_type = v_new_item.origin_document_type,
      origin_document_no = v_new_item.origin_document_no,
      origin_document_date = v_new_item.origin_document_date,
      allocation_purpose = v_new_item.allocation_purpose,
      allocation_proposal_type = v_new_item.allocation_proposal_type,
      allocation_proposal_no = v_new_item.allocation_proposal_no,
      allocation_proposal_date = v_new_item.allocation_proposal_date,
      allocation_approval_type = v_new_item.allocation_approval_type,
      allocation_approval_no = v_new_item.allocation_approval_no,
      allocation_approval_date = v_new_item.allocation_approval_date,
      exit_document_no = v_new_item.exit_document_no,
      exit_document_date = v_new_item.exit_document_date,
      exit_type = v_new_item.exit_type,
      exit_notes = v_new_item.exit_notes
  where id = v_item.id
  returning * into v_new_item;

  for v_patch in
    select value from jsonb_array_elements(coalesce(p_event_patches, '[]'::jsonb))
  loop
    select * into v_event
    from public.events
    where id = nullif(v_patch->>'id', '')::uuid
      and inventory_id = v_item.id
    for update;
    if not found then
      raise exception 'not found: event' using errcode = 'P0002';
    end if;
    select * into v_new_event
    from jsonb_populate_record(v_event, v_patch - 'id');
    if coalesce(btrim(v_new_event.label), '') = '' then
      raise exception 'event label cannot be empty' using errcode = '22023';
    end if;
    update public.events
    set label = v_new_event.label,
        document_no = v_new_event.document_no,
        document_date = v_new_event.document_date,
        notes = v_new_event.notes
    where id = v_event.id;
  end loop;

  for v_patch in
    select value from jsonb_array_elements(coalesce(p_process_patches, '[]'::jsonb))
  loop
    select * into v_process
    from public.dispositions
    where id = nullif(v_patch->>'id', '')::uuid
      and inventory_id = v_item.id
    for update;
    if not found then
      raise exception 'not found: disposition' using errcode = 'P0002';
    end if;
    select * into v_new_process
    from jsonb_populate_record(v_process, v_patch - 'id');
    if v_new_process.sale_value < 0 or v_new_process.htl_value < 0 or v_new_process.destruction_cost < 0 then
      raise exception 'negative process value' using errcode = '22023';
    end if;
    if v_new_process.execution_start_date is not null
       and v_new_process.execution_end_date is not null
       and v_new_process.execution_end_date < v_new_process.execution_start_date then
      raise exception 'invalid execution date range' using errcode = '22023';
    end if;
    update public.dispositions
    set proposal_type = v_new_process.proposal_type,
        recipient_code = v_new_process.recipient_code,
        recipient_name = v_new_process.recipient_name,
        sale_value = v_new_process.sale_value,
        htl_value = v_new_process.htl_value,
        execution_start_date = v_new_process.execution_start_date,
        execution_end_date = v_new_process.execution_end_date,
        schedule_document_no = v_new_process.schedule_document_no,
        schedule_document_date = v_new_process.schedule_document_date,
        auction_outcome = v_new_process.auction_outcome,
        allocation_target = v_new_process.allocation_target,
        destruction_cost = v_new_process.destruction_cost,
        transfer_type = v_new_process.transfer_type
    where id = v_process.id;
  end loop;

  v_notes := 'Data barang diperbarui melalui rekonsiliasi. Alasan perubahan: ' || p_reason || '.';

  insert into public.events (
    inventory_id, code, label, notes, actor, document_id
  ) values (
    v_item.id, 'perubahan_data_barang', 'Perubahan data barang',
    v_notes, btrim(p_actor), p_document_id
  );

  insert into public.reconciliations (
    reconciliation_type, action, inventory_id, inventory_reference, inventory_type,
    previous_status_code, previous_status_label, result_status_code, result_status_label,
    notes, actor
  ) values (
    'data_correction', 'updated', v_item.id, v_new_item.reference_no, v_new_item.item_type,
    v_item.status_code, v_item.status_label, v_new_item.status_code, v_new_item.status_label,
    v_notes, btrim(p_actor)
  ) returning * into v_record;

  return jsonb_build_object(
    'record', to_jsonb(v_record),
    'item', to_jsonb(v_new_item) - 'search_text'
  );
end;
$$;

revoke all on function public.livira_create_disposition(uuid,text,text,text,timestamptz) from anon, authenticated;
grant execute on function public.livira_create_disposition(uuid,text,text,text,timestamptz) to service_role;
revoke all on function public.livira_correct_inventory_data(uuid,text,text,jsonb,jsonb,jsonb,uuid,timestamptz) from anon, authenticated;
grant execute on function public.livira_correct_inventory_data(uuid,text,text,jsonb,jsonb,jsonb,uuid,timestamptz) to service_role;

commit;

-- ============================================================================
-- END MIGRATION: 017_transfer_lelang_rekonsiliasi_perubahan_data.sql
-- ============================================================================

-- ============================================================================
-- BEGIN MIGRATION: 018_reconciliation_tabs_change_audit_reports.sql
-- ============================================================================

-- LIVIRA migration 018
-- 1. Memisahkan hasil rekonsiliasi fisik dan perubahan data barang.
-- 2. Menyimpan rincian setiap nilai sebelum dan sesudah perubahan.
-- 3. Menjamin laporan serta ekspor perubahan data berasal dari audit yang sama.

begin;

alter table public.reconciliations
  add column if not exists correction_reason text not null default '';

alter table public.reconciliations
  add column if not exists change_details jsonb not null default '[]'::jsonb;

alter table public.reconciliations
  drop constraint if exists reconciliations_change_details_array_check;
alter table public.reconciliations
  add constraint reconciliations_change_details_array_check
  check (jsonb_typeof(change_details) = 'array');

update public.reconciliations
set correction_reason = case
  when notes ilike '%Alasan perubahan: Kesalahan input.%' then 'Kesalahan input'
  when notes ilike '%Alasan perubahan: Error pada saat pengisian awal.%' then 'Error pada saat pengisian awal'
  else correction_reason
end
where reconciliation_type = 'data_correction'
  and correction_reason = '';

create or replace function public.livira_correct_inventory_data(
  p_inventory_id uuid,
  p_actor text,
  p_reason text,
  p_item_patch jsonb,
  p_event_patches jsonb default '[]'::jsonb,
  p_process_patches jsonb default '[]'::jsonb,
  p_document_id uuid default null,
  p_expected_updated_at timestamptz default null
)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_item public.inventory_items%rowtype;
  v_new_item public.inventory_items%rowtype;
  v_event public.events%rowtype;
  v_new_event public.events%rowtype;
  v_process public.dispositions%rowtype;
  v_new_process public.dispositions%rowtype;
  v_record public.reconciliations%rowtype;
  v_patch jsonb;
  v_notes text;
  v_field text;
  v_before text;
  v_after text;
  v_context text;
  v_change_details jsonb := '[]'::jsonb;
begin
  if coalesce(btrim(p_actor), '') = '' then
    raise exception 'invalid actor' using errcode = '22023';
  end if;
  if p_reason not in ('Kesalahan input', 'Error pada saat pengisian awal') then
    raise exception 'invalid correction reason' using errcode = '22023';
  end if;
  if jsonb_typeof(coalesce(p_item_patch, '{}'::jsonb)) <> 'object'
     or jsonb_typeof(coalesce(p_event_patches, '[]'::jsonb)) <> 'array'
     or jsonb_typeof(coalesce(p_process_patches, '[]'::jsonb)) <> 'array' then
    raise exception 'invalid correction payload' using errcode = '22023';
  end if;

  select * into v_item
  from public.inventory_items
  where id = p_inventory_id
  for update;
  if not found then
    raise exception 'not found: inventory' using errcode = 'P0002';
  end if;
  if p_expected_updated_at is not null and v_item.updated_at is distinct from p_expected_updated_at then
    raise exception 'record changed by another user' using errcode = '40001';
  end if;

  select * into v_new_item
  from jsonb_populate_record(v_item, coalesce(p_item_patch, '{}'::jsonb));

  if coalesce(btrim(v_new_item.reference_no), '') = ''
     or coalesce(btrim(v_new_item.determination_no), '') = ''
     or v_new_item.determination_date is null
     or coalesce(btrim(v_new_item.description), '') = '' then
    raise exception 'required inventory data is empty' using errcode = '22023';
  end if;

  if v_new_item.facility_id is not null then
    select name into v_new_item.facility_name
    from public.facilities
    where id = v_new_item.facility_id and active = true;
    if not found then
      raise exception 'invalid facility' using errcode = '22023';
    end if;
  else
    v_new_item.facility_name := '';
  end if;

  foreach v_field in array array[
    'reference_no','item_type','origin_type','manifest_no','manifest_date','manifest_position',
    'determination_no','determination_date','category','entrusted_category','source_office',
    'description','item_kind','quantity','unit','goods_value','goods_condition',
    'location','location_status','at_tpp','owner_name','owner_address','origin_warehouse',
    'facility_id','facility_name','load_type','container_no','container_size','estimated_volume_m3',
    'physical_unit_id','occupancy_primary','pfpd_required','research_request_no','research_request_date',
    'hs_code','is_restricted','restriction_rule','origin_document_type','origin_document_no',
    'origin_document_date','allocation_purpose','allocation_proposal_type','allocation_proposal_no',
    'allocation_proposal_date','allocation_approval_type','allocation_approval_no','allocation_approval_date',
    'exit_document_no','exit_document_date','exit_type','exit_notes'
  ]
  loop
    v_before := coalesce(to_jsonb(v_item)->>v_field, '');
    v_after := coalesce(to_jsonb(v_new_item)->>v_field, '');
    if v_before is distinct from v_after then
      v_change_details := v_change_details || jsonb_build_array(jsonb_build_object(
        'section', 'inventory',
        'field', v_field,
        'before', v_before,
        'after', v_after
      ));
    end if;
  end loop;

  update public.inventory_items
  set reference_no = v_new_item.reference_no,
      item_type = v_new_item.item_type,
      origin_type = v_new_item.origin_type,
      manifest_no = v_new_item.manifest_no,
      manifest_date = v_new_item.manifest_date,
      manifest_position = v_new_item.manifest_position,
      determination_no = v_new_item.determination_no,
      determination_date = v_new_item.determination_date,
      category = v_new_item.category,
      entrusted_category = v_new_item.entrusted_category,
      source_office = v_new_item.source_office,
      description = v_new_item.description,
      item_kind = v_new_item.item_kind,
      quantity = v_new_item.quantity,
      unit = v_new_item.unit,
      goods_value = v_new_item.goods_value,
      goods_condition = v_new_item.goods_condition,
      location = v_new_item.location,
      location_status = v_new_item.location_status,
      at_tpp = v_new_item.at_tpp,
      owner_name = v_new_item.owner_name,
      owner_address = v_new_item.owner_address,
      origin_warehouse = v_new_item.origin_warehouse,
      facility_id = v_new_item.facility_id,
      facility_name = v_new_item.facility_name,
      load_type = v_new_item.load_type,
      container_no = v_new_item.container_no,
      container_size = v_new_item.container_size,
      estimated_volume_m3 = v_new_item.estimated_volume_m3,
      physical_unit_id = v_new_item.physical_unit_id,
      occupancy_primary = v_new_item.occupancy_primary,
      pfpd_required = v_new_item.pfpd_required,
      research_request_no = v_new_item.research_request_no,
      research_request_date = v_new_item.research_request_date,
      hs_code = v_new_item.hs_code,
      is_restricted = v_new_item.is_restricted,
      restriction_rule = v_new_item.restriction_rule,
      origin_document_type = v_new_item.origin_document_type,
      origin_document_no = v_new_item.origin_document_no,
      origin_document_date = v_new_item.origin_document_date,
      allocation_purpose = v_new_item.allocation_purpose,
      allocation_proposal_type = v_new_item.allocation_proposal_type,
      allocation_proposal_no = v_new_item.allocation_proposal_no,
      allocation_proposal_date = v_new_item.allocation_proposal_date,
      allocation_approval_type = v_new_item.allocation_approval_type,
      allocation_approval_no = v_new_item.allocation_approval_no,
      allocation_approval_date = v_new_item.allocation_approval_date,
      exit_document_no = v_new_item.exit_document_no,
      exit_document_date = v_new_item.exit_document_date,
      exit_type = v_new_item.exit_type,
      exit_notes = v_new_item.exit_notes
  where id = v_item.id
  returning * into v_new_item;

  for v_patch in
    select value from jsonb_array_elements(coalesce(p_event_patches, '[]'::jsonb))
  loop
    select * into v_event
    from public.events
    where id = nullif(v_patch->>'id', '')::uuid
      and inventory_id = v_item.id
    for update;
    if not found then
      raise exception 'not found: event' using errcode = 'P0002';
    end if;
    select * into v_new_event
    from jsonb_populate_record(v_event, v_patch - 'id');
    if coalesce(btrim(v_new_event.label), '') = '' then
      raise exception 'event label cannot be empty' using errcode = '22023';
    end if;
    v_context := coalesce(nullif(btrim(v_event.label), ''), 'Tahapan timeline');
    foreach v_field in array array['label','document_no','document_date','notes']
    loop
      v_before := coalesce(to_jsonb(v_event)->>v_field, '');
      v_after := coalesce(to_jsonb(v_new_event)->>v_field, '');
      if v_before is distinct from v_after then
        v_change_details := v_change_details || jsonb_build_array(jsonb_build_object(
          'section', 'timeline',
          'entity_id', v_event.id::text,
          'context', v_context,
          'field', v_field,
          'before', v_before,
          'after', v_after
        ));
      end if;
    end loop;
    update public.events
    set label = v_new_event.label,
        document_no = v_new_event.document_no,
        document_date = v_new_event.document_date,
        notes = v_new_event.notes
    where id = v_event.id;
  end loop;

  for v_patch in
    select value from jsonb_array_elements(coalesce(p_process_patches, '[]'::jsonb))
  loop
    select * into v_process
    from public.dispositions
    where id = nullif(v_patch->>'id', '')::uuid
      and inventory_id = v_item.id
    for update;
    if not found then
      raise exception 'not found: disposition' using errcode = 'P0002';
    end if;
    select * into v_new_process
    from jsonb_populate_record(v_process, v_patch - 'id');
    if v_new_process.sale_value < 0 or v_new_process.htl_value < 0 or v_new_process.destruction_cost < 0 then
      raise exception 'negative process value' using errcode = '22023';
    end if;
    if v_new_process.execution_start_date is not null
       and v_new_process.execution_end_date is not null
       and v_new_process.execution_end_date < v_new_process.execution_start_date then
      raise exception 'invalid execution date range' using errcode = '22023';
    end if;
    v_context := coalesce(nullif(btrim(v_process.status_label), ''), upper(v_process.disposition_type));
    foreach v_field in array array[
      'proposal_type','recipient_code','recipient_name','sale_value','htl_value',
      'execution_start_date','execution_end_date','schedule_document_no','schedule_document_date',
      'auction_outcome','allocation_target','destruction_cost','transfer_type'
    ]
    loop
      v_before := coalesce(to_jsonb(v_process)->>v_field, '');
      v_after := coalesce(to_jsonb(v_new_process)->>v_field, '');
      if v_before is distinct from v_after then
        v_change_details := v_change_details || jsonb_build_array(jsonb_build_object(
          'section', 'process',
          'entity_id', v_process.id::text,
          'context', v_context,
          'field', v_field,
          'before', v_before,
          'after', v_after
        ));
      end if;
    end loop;
    update public.dispositions
    set proposal_type = v_new_process.proposal_type,
        recipient_code = v_new_process.recipient_code,
        recipient_name = v_new_process.recipient_name,
        sale_value = v_new_process.sale_value,
        htl_value = v_new_process.htl_value,
        execution_start_date = v_new_process.execution_start_date,
        execution_end_date = v_new_process.execution_end_date,
        schedule_document_no = v_new_process.schedule_document_no,
        schedule_document_date = v_new_process.schedule_document_date,
        auction_outcome = v_new_process.auction_outcome,
        allocation_target = v_new_process.allocation_target,
        destruction_cost = v_new_process.destruction_cost,
        transfer_type = v_new_process.transfer_type
    where id = v_process.id;
  end loop;

  if jsonb_array_length(v_change_details) = 0 then
    raise exception 'no data changes detected' using errcode = '22023';
  end if;

  v_notes := 'Data barang diperbarui melalui rekonsiliasi. Alasan perubahan: ' || p_reason || '.';

  insert into public.events (
    inventory_id, code, label, notes, actor, document_id
  ) values (
    v_item.id, 'perubahan_data_barang', 'Perubahan data barang',
    v_notes, btrim(p_actor), p_document_id
  );

  insert into public.reconciliations (
    reconciliation_type, action, inventory_id, inventory_reference, inventory_type,
    previous_status_code, previous_status_label, result_status_code, result_status_label,
    correction_reason, change_details, notes, actor
  ) values (
    'data_correction', 'updated', v_item.id, v_new_item.reference_no, v_new_item.item_type,
    v_item.status_code, v_item.status_label, v_new_item.status_code, v_new_item.status_label,
    p_reason, v_change_details, v_notes, btrim(p_actor)
  ) returning * into v_record;

  return jsonb_build_object(
    'record', to_jsonb(v_record),
    'item', to_jsonb(v_new_item) - 'search_text'
  );
end;
$$;

revoke all on function public.livira_correct_inventory_data(uuid,text,text,jsonb,jsonb,jsonb,uuid,timestamptz) from anon, authenticated;
grant execute on function public.livira_correct_inventory_data(uuid,text,text,jsonb,jsonb,jsonb,uuid,timestamptz) to service_role;

commit;

-- ============================================================================
-- END MIGRATION: 018_reconciliation_tabs_change_audit_reports.sql
-- ============================================================================

-- ============================================================================
-- VERIFIKASI SETUP
-- Hasil yang diharapkan:
--   inventory_count = 0
--   disposition_count = 0
--   event_count = 0
--   role_count > 0
--   parameter_count > 0
--   facility_count = 4
-- ============================================================================
select
  (select count(*) from public.inventory_items) as inventory_count,
  (select count(*) from public.dispositions) as disposition_count,
  (select count(*) from public.events) as event_count,
  (select count(*) from public.app_roles) as role_count,
  (select count(*) from public.app_parameters) as parameter_count,
  (select count(*) from public.facilities) as facility_count,
  (select count(*) from storage.buckets where id = 'livira-documents') as private_document_bucket_count;

-- =====================================================================
-- MIGRATION 019
-- =====================================================================

-- LIVIRA migration 019
-- Dashboard document/FCL/LCL metrics, BL number for BTD, optional census quantity detail,
-- BTD report support, and multi-goods-per-container upload compatibility.

begin;

alter table public.inventory_items
  add column if not exists bl_no text not null default '',
  add column if not exists quantity_detail text not null default '';

comment on column public.inventory_items.bl_no is 'Nomor bill of lading; wajib pada pencatatan BTD baru di aplikasi.';
comment on column public.inventory_items.quantity_detail is 'Rincian tekstual jumlah barang hasil pencacahan; opsional.';

create or replace function public.livira_inventory_search_text(i public.inventory_items)
returns text
language sql
immutable
set search_path = public, pg_temp
as $$
  select lower(concat_ws(' ',
    i.reference_no, i.item_type, i.origin_type,
    i.bl_no, i.manifest_no, i.manifest_position,
    i.determination_no, i.category, i.entrusted_category, i.source_office,
    i.description, i.item_kind, i.goods_condition,
    i.quantity::text, i.quantity_detail, i.unit, i.goods_value::text,
    i.location, i.location_status, i.owner_name, i.owner_address,
    i.origin_warehouse, i.facility_id, i.facility_name,
    i.load_type, i.container_no, i.container_size,
    i.estimated_volume_m3::text, i.physical_unit_id,
    i.research_request_no, i.hs_code, i.restriction_rule,
    i.origin_document_type, i.origin_document_no,
    i.allocation_purpose, i.allocation_proposal_type,
    i.allocation_proposal_no, i.allocation_approval_type,
    i.allocation_approval_no, i.exit_document_no, i.exit_type,
    i.exit_notes, i.status_code, i.status_label,
    i.current_disposition
  ));
$$;


update public.inventory_items i
set search_text = public.livira_inventory_search_text(i)
where search_text is distinct from public.livira_inventory_search_text(i);

create or replace function public.livira_inventory_dashboard_metrics(p_type text default null)
returns jsonb
language sql
stable
security definer
set search_path = public, pg_temp
as $$
with scoped as (
  select *
  from public.inventory_items
  where is_active = true
    and (p_type is null or item_type = p_type)
)
select jsonb_build_object(
  'documents', count(distinct concat_ws('|',
    item_type,
    upper(trim(coalesce(nullif(determination_no, ''), reference_no))),
    coalesce(determination_date::date::text, '')
  ))::int,
  'fcl', count(distinct coalesce(
    nullif(regexp_replace(upper(container_no), '[^A-Z0-9]', '', 'g'), ''),
    nullif(trim(physical_unit_id), '')
  )) filter (where upper(trim(load_type)) = 'FCL')::int,
  'lcl', count(distinct coalesce(
    nullif(trim(physical_unit_id), ''),
    concat_ws('|', item_type, upper(trim(coalesce(nullif(determination_no, ''), reference_no))), coalesce(determination_date::date::text, ''))
  )) filter (where upper(trim(load_type)) = 'LCL')::int
)
from scoped;
$$;


create or replace function public.livira_dashboard_summary()
returns jsonb
language sql
stable
security definer
set search_path = public, pg_temp
as $$
with active_items as (
  select *
  from public.inventory_items
  where is_active = true
),
eligible_occupancy as (
  select distinct on (facility_id, unit_key)
    facility_id,
    case
      when upper(load_type) = 'FCL' then
        case upper(container_size)
          when '40' then 2::numeric
          when '40HC' then 2::numeric
          when '45' then 2.25::numeric
          when '45HC' then 2.25::numeric
          else 1::numeric
        end
      else 0::numeric
    end as yard_used,
    case
      when upper(load_type) = 'LCL' and estimated_volume_m3 > 0 then estimated_volume_m3
      else 0::numeric
    end as shed_used
  from (
    select ai.*,
      case
        when trim(ai.physical_unit_id) <> '' then trim(ai.physical_unit_id)
        when upper(ai.load_type) = 'FCL' and trim(ai.container_no) <> ''
          then 'FCL:' || upper(regexp_replace(ai.container_no, '[ .-]', '', 'g'))
        else 'ITEM:' || ai.id::text
      end as unit_key
    from active_items ai
    where ai.at_tpp = true
      and ai.facility_id is not null
      and (trim(ai.physical_unit_id) = '' or ai.occupancy_primary = true)
  ) candidates
  order by facility_id, unit_key, updated_at desc
),
facility_rows as (
  select
    f.id as facility_id,
    f.name as facility_name,
    count(ai.id) filter (where ai.item_type = 'BTD')::int as btd,
    count(ai.id) filter (where ai.item_type = 'BDN')::int as bdn,
    count(ai.id) filter (where ai.item_type = 'BMMN')::int as bmmn,
    count(ai.id)::int as total,
    f.yard_capacity,
    coalesce((select sum(eo.yard_used) from eligible_occupancy eo where eo.facility_id = f.id), 0) as yard_used,
    f.shed_capacity,
    coalesce((select sum(eo.shed_used) from eligible_occupancy eo where eo.facility_id = f.id), 0) as shed_used,
    f.sort_order
  from public.facilities f
  left join active_items ai on ai.facility_id = f.id
  where f.active = true
  group by f.id, f.name, f.yard_capacity, f.shed_capacity, f.sort_order
),
process_counts as (
  select
    count(*) filter (where is_active and disposition_type = 'lelang')::int as auction_active,
    count(*) filter (where is_active and disposition_type = 'musnah')::int as destruction_active,
    count(*) filter (where is_active and disposition_type = 'hibah')::int as grant_active,
    count(*) filter (
      where not is_active
        and date_trunc('month', updated_at) = date_trunc('month', now())
    )::int as completed_this_month
  from public.dispositions
),
attention as (
  select coalesce(jsonb_agg(to_jsonb(q) - 'search_text' order by q.determination_date asc), '[]'::jsonb) as rows
  from (
    select *
    from active_items
    where coalesce(determination_date, created_at) <= now() - interval '45 days'
    order by determination_date asc nulls first, created_at asc
    limit 5
  ) q
),
recent as (
  select coalesce(jsonb_agg(to_jsonb(q) order by q.created_at desc), '[]'::jsonb) as rows
  from (
    select *
    from public.events
    order by created_at desc
    limit 6
  ) q
),
item_counts as (
  select
    count(*)::int as active_total,
    count(*) filter (where item_type = 'BTD')::int as btd_total,
    count(*) filter (where item_type = 'BDN')::int as bdn_total,
    count(*) filter (where item_type = 'BMMN')::int as bmmn_total
  from active_items
),
facility_json as (
  select
    coalesce(jsonb_agg(
      jsonb_build_object(
        'facility_id', facility_id,
        'facility_name', facility_name,
        'btd', btd,
        'bdn', bdn,
        'bmmn', bmmn,
        'total', total,
        'yard_capacity', yard_capacity,
        'yard_used', yard_used,
        'shed_capacity', shed_capacity,
        'shed_used', shed_used
      ) order by sort_order, facility_name
    ), '[]'::jsonb) as rows,
    coalesce(sum(yard_capacity), 0) as yard_capacity,
    coalesce(sum(yard_used), 0) as yard_used,
    coalesce(sum(shed_capacity), 0) as shed_capacity,
    coalesce(sum(shed_used), 0) as shed_used
  from facility_rows
)
select jsonb_build_object(
  'active_total', ic.active_total,
  'btd_total', ic.btd_total,
  'bdn_total', ic.bdn_total,
  'bmmn_total', ic.bmmn_total,
  'active_summary', public.livira_inventory_dashboard_metrics(null),
  'btd_summary', public.livira_inventory_dashboard_metrics('BTD'),
  'bdn_summary', public.livira_inventory_dashboard_metrics('BDN'),
  'bmmn_summary', public.livira_inventory_dashboard_metrics('BMMN'),
  'auction_active', pc.auction_active,
  'destruction_active', pc.destruction_active,
  'grant_active', pc.grant_active,
  'completed_this_month', pc.completed_this_month,
  'occupancy', jsonb_build_object(
    'yard_capacity', fj.yard_capacity,
    'yard_used', fj.yard_used,
    'shed_capacity', fj.shed_capacity,
    'shed_used', fj.shed_used
  ),
  'facility_breakdown', fj.rows,
  'recent_events', r.rows,
  'attention_items', a.rows
)
from item_counts ic
cross join process_counts pc
cross join facility_json fj
cross join recent r
cross join attention a;
$$;


create or replace function public.livira_create_inventories(p_rows jsonb)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_entry jsonb;
  v_item public.inventory_items%rowtype;
  v_process_id uuid;
  v_initial_type text;
  v_active boolean;
  v_result jsonb := '[]'::jsonb;
begin
  if jsonb_typeof(p_rows) <> 'array'
     or jsonb_array_length(p_rows) = 0
     or jsonb_array_length(p_rows) > 500 then
    raise exception 'invalid transition: inventory batch size' using errcode = '22023';
  end if;

  for v_entry in select value from jsonb_array_elements(p_rows)
  loop
    insert into public.inventory_items (
      reference_no, item_type, origin_type,
      bl_no, manifest_no, manifest_date, manifest_position,
      determination_no, determination_date, category,
      entrusted_category, source_office,
      description, item_kind, quantity, quantity_detail, unit, goods_value, goods_condition,
      location, location_status, at_tpp, owner_name, owner_address,
      origin_warehouse, facility_id, facility_name,
      load_type, container_no, container_size, estimated_volume_m3,
      physical_unit_id, occupancy_primary, pfpd_required,
      restriction_rule, status_code, status_label, current_disposition,
      is_active, created_by
    ) values (
      coalesce(v_entry->>'reference_no', ''),
      coalesce(v_entry->>'item_type', ''),
      coalesce(v_entry->>'origin_type', ''),
      coalesce(v_entry->>'bl_no', ''),
      coalesce(v_entry->>'manifest_no', ''),
      nullif(v_entry->>'manifest_date', '')::timestamptz,
      coalesce(v_entry->>'manifest_position', ''),
      coalesce(v_entry->>'determination_no', ''),
      nullif(v_entry->>'determination_date', '')::timestamptz,
      coalesce(v_entry->>'category', ''),
      coalesce(v_entry->>'entrusted_category', ''),
      coalesce(v_entry->>'source_office', ''),
      coalesce(v_entry->>'description', ''),
      coalesce(v_entry->>'item_kind', ''),
      coalesce(nullif(v_entry->>'quantity', '')::numeric, 0),
      coalesce(v_entry->>'quantity_detail', ''),
      coalesce(v_entry->>'unit', ''),
      coalesce(nullif(v_entry->>'goods_value', '')::bigint, 0),
      coalesce(v_entry->>'goods_condition', ''),
      coalesce(v_entry->>'location', ''),
      coalesce(v_entry->>'location_status', ''),
      coalesce(nullif(v_entry->>'at_tpp', '')::boolean, false),
      coalesce(v_entry->>'owner_name', ''),
      coalesce(v_entry->>'owner_address', ''),
      coalesce(v_entry->>'origin_warehouse', ''),
      nullif(v_entry->>'facility_id', ''),
      coalesce(v_entry->>'facility_name', ''),
      coalesce(v_entry->>'load_type', ''),
      coalesce(v_entry->>'container_no', ''),
      coalesce(v_entry->>'container_size', ''),
      coalesce(nullif(v_entry->>'estimated_volume_m3', '')::numeric, 0),
      coalesce(v_entry->>'physical_unit_id', ''),
      coalesce(nullif(v_entry->>'occupancy_primary', '')::boolean, true),
      coalesce(nullif(v_entry->>'pfpd_required', '')::boolean, true),
      coalesce(v_entry->>'restriction_rule', ''),
      coalesce(v_entry->>'status_code', ''),
      coalesce(v_entry->>'status_label', ''),
      null,
      coalesce(nullif(v_entry->>'is_active', '')::boolean, true),
      coalesce(v_entry->>'created_by', '')
    ) returning * into v_item;

    v_process_id := null;
    v_initial_type := nullif(trim(coalesce(v_entry->>'_initial_disposition_type', '')), '');
    if v_initial_type is not null then
      if v_initial_type not in ('lelang', 'musnah', 'hibah') then
        raise exception 'invalid disposition type' using errcode = '22023';
      end if;
      v_active := coalesce(v_entry->>'_initial_status_code', '') not in (
        'alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima'
      );
      insert into public.dispositions (
        inventory_id, disposition_type, round, status_code, status_label,
        schedule_document_no, schedule_document_date, transfer_type,
        is_active, created_by
      ) values (
        v_item.id, v_initial_type, 1, v_item.status_code, v_item.status_label,
        case when coalesce(v_entry->>'_initial_status_code', '') = 'jadwal_lelang' then v_item.determination_no else '' end,
        case when coalesce(v_entry->>'_initial_status_code', '') = 'jadwal_lelang' then v_item.determination_date else null end,
        coalesce(v_entry->>'_initial_transfer_type', ''),
        v_active, v_item.created_by
      ) returning id into v_process_id;

      if v_active then
        update public.inventory_items
        set current_disposition = v_initial_type
        where id = v_item.id
        returning * into v_item;
      end if;
    end if;

    insert into public.events (
      inventory_id, disposition_id, disposition_type, code, label,
      document_no, document_date, notes, actor, document_id
    ) values (
      v_item.id, v_process_id, v_initial_type, v_item.status_code, v_item.status_label,
      v_item.determination_no, v_item.determination_date,
      case when coalesce(nullif(v_entry->>'_reconciliation_created', ''), 'false')::boolean
        then 'Inventory ditambahkan melalui rekonsiliasi kondisi fisik.' else '' end,
      v_item.created_by,
      nullif(v_entry->>'_document_id', '')::uuid
    );

    v_result := v_result || jsonb_build_array(to_jsonb(v_item) - 'search_text');
  end loop;

  return v_result;
end;
$$;


revoke all on function public.livira_inventory_dashboard_metrics(text) from public, anon, authenticated;
revoke all on function public.livira_dashboard_summary() from public, anon, authenticated;
revoke all on function public.livira_create_inventories(jsonb) from public, anon, authenticated;
grant execute on function public.livira_inventory_dashboard_metrics(text) to service_role;
grant execute on function public.livira_dashboard_summary() to service_role;
grant execute on function public.livira_create_inventories(jsonb) to service_role;

commit;


-- LIVIRA migration 020
-- Mandatory BL date for new BTD records, including manual entry, Excel import,
-- reporting, search, correction, and batch inventory creation.

begin;

alter table public.inventory_items
  add column if not exists bl_date timestamptz;

comment on column public.inventory_items.bl_date is
  'Tanggal bill of lading; wajib pada pencatatan BTD baru di aplikasi. Data lama boleh tetap kosong.';

create or replace function public.livira_inventory_search_text(i public.inventory_items)
returns text
language sql
immutable
set search_path = public, pg_temp
as $$
  select lower(concat_ws(' ',
    i.reference_no, i.item_type, i.origin_type,
    i.bl_no, coalesce(i.bl_date::date::text, ''), i.manifest_no, i.manifest_position,
    i.determination_no, i.category, i.entrusted_category, i.source_office,
    i.description, i.item_kind, i.goods_condition,
    i.quantity::text, i.quantity_detail, i.unit, i.goods_value::text,
    i.location, i.location_status, i.owner_name, i.owner_address,
    i.origin_warehouse, i.facility_id, i.facility_name,
    i.load_type, i.container_no, i.container_size,
    i.estimated_volume_m3::text, i.physical_unit_id,
    i.research_request_no, i.hs_code, i.restriction_rule,
    i.origin_document_type, i.origin_document_no,
    i.allocation_purpose, i.allocation_proposal_type,
    i.allocation_proposal_no, i.allocation_approval_type,
    i.allocation_approval_no, i.exit_document_no, i.exit_type,
    i.exit_notes, i.status_code, i.status_label,
    i.current_disposition
  ));
$$;


update public.inventory_items i
set search_text = public.livira_inventory_search_text(i)
where search_text is distinct from public.livira_inventory_search_text(i);

create or replace function public.livira_create_inventories(p_rows jsonb)
returns jsonb
language plpgsql
security definer
set search_path = public, pg_temp
as $$
declare
  v_entry jsonb;
  v_item public.inventory_items%rowtype;
  v_process_id uuid;
  v_initial_type text;
  v_active boolean;
  v_result jsonb := '[]'::jsonb;
begin
  if jsonb_typeof(p_rows) <> 'array'
     or jsonb_array_length(p_rows) = 0
     or jsonb_array_length(p_rows) > 500 then
    raise exception 'invalid transition: inventory batch size' using errcode = '22023';
  end if;

  for v_entry in select value from jsonb_array_elements(p_rows)
  loop
    insert into public.inventory_items (
      reference_no, item_type, origin_type,
      bl_no, bl_date, manifest_no, manifest_date, manifest_position,
      determination_no, determination_date, category,
      entrusted_category, source_office,
      description, item_kind, quantity, quantity_detail, unit, goods_value, goods_condition,
      location, location_status, at_tpp, owner_name, owner_address,
      origin_warehouse, facility_id, facility_name,
      load_type, container_no, container_size, estimated_volume_m3,
      physical_unit_id, occupancy_primary, pfpd_required,
      restriction_rule, status_code, status_label, current_disposition,
      is_active, created_by
    ) values (
      coalesce(v_entry->>'reference_no', ''),
      coalesce(v_entry->>'item_type', ''),
      coalesce(v_entry->>'origin_type', ''),
      coalesce(v_entry->>'bl_no', ''),
      nullif(v_entry->>'bl_date', '')::timestamptz,
      coalesce(v_entry->>'manifest_no', ''),
      nullif(v_entry->>'manifest_date', '')::timestamptz,
      coalesce(v_entry->>'manifest_position', ''),
      coalesce(v_entry->>'determination_no', ''),
      nullif(v_entry->>'determination_date', '')::timestamptz,
      coalesce(v_entry->>'category', ''),
      coalesce(v_entry->>'entrusted_category', ''),
      coalesce(v_entry->>'source_office', ''),
      coalesce(v_entry->>'description', ''),
      coalesce(v_entry->>'item_kind', ''),
      coalesce(nullif(v_entry->>'quantity', '')::numeric, 0),
      coalesce(v_entry->>'quantity_detail', ''),
      coalesce(v_entry->>'unit', ''),
      coalesce(nullif(v_entry->>'goods_value', '')::bigint, 0),
      coalesce(v_entry->>'goods_condition', ''),
      coalesce(v_entry->>'location', ''),
      coalesce(v_entry->>'location_status', ''),
      coalesce(nullif(v_entry->>'at_tpp', '')::boolean, false),
      coalesce(v_entry->>'owner_name', ''),
      coalesce(v_entry->>'owner_address', ''),
      coalesce(v_entry->>'origin_warehouse', ''),
      nullif(v_entry->>'facility_id', ''),
      coalesce(v_entry->>'facility_name', ''),
      coalesce(v_entry->>'load_type', ''),
      coalesce(v_entry->>'container_no', ''),
      coalesce(v_entry->>'container_size', ''),
      coalesce(nullif(v_entry->>'estimated_volume_m3', '')::numeric, 0),
      coalesce(v_entry->>'physical_unit_id', ''),
      coalesce(nullif(v_entry->>'occupancy_primary', '')::boolean, true),
      coalesce(nullif(v_entry->>'pfpd_required', '')::boolean, true),
      coalesce(v_entry->>'restriction_rule', ''),
      coalesce(v_entry->>'status_code', ''),
      coalesce(v_entry->>'status_label', ''),
      null,
      coalesce(nullif(v_entry->>'is_active', '')::boolean, true),
      coalesce(v_entry->>'created_by', '')
    ) returning * into v_item;

    v_process_id := null;
    v_initial_type := nullif(trim(coalesce(v_entry->>'_initial_disposition_type', '')), '');
    if v_initial_type is not null then
      if v_initial_type not in ('lelang', 'musnah', 'hibah') then
        raise exception 'invalid disposition type' using errcode = '22023';
      end if;
      v_active := coalesce(v_entry->>'_initial_status_code', '') not in (
        'alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima'
      );
      insert into public.dispositions (
        inventory_id, disposition_type, round, status_code, status_label,
        schedule_document_no, schedule_document_date, transfer_type,
        is_active, created_by
      ) values (
        v_item.id, v_initial_type, 1, v_item.status_code, v_item.status_label,
        case when coalesce(v_entry->>'_initial_status_code', '') = 'jadwal_lelang' then v_item.determination_no else '' end,
        case when coalesce(v_entry->>'_initial_status_code', '') = 'jadwal_lelang' then v_item.determination_date else null end,
        coalesce(v_entry->>'_initial_transfer_type', ''),
        v_active, v_item.created_by
      ) returning id into v_process_id;

      if v_active then
        update public.inventory_items
        set current_disposition = v_initial_type
        where id = v_item.id
        returning * into v_item;
      end if;
    end if;

    insert into public.events (
      inventory_id, disposition_id, disposition_type, code, label,
      document_no, document_date, notes, actor, document_id
    ) values (
      v_item.id, v_process_id, v_initial_type, v_item.status_code, v_item.status_label,
      v_item.determination_no, v_item.determination_date,
      case when coalesce(nullif(v_entry->>'_reconciliation_created', ''), 'false')::boolean
        then 'Inventory ditambahkan melalui rekonsiliasi kondisi fisik.' else '' end,
      v_item.created_by,
      nullif(v_entry->>'_document_id', '')::uuid
    );

    v_result := v_result || jsonb_build_array(to_jsonb(v_item) - 'search_text');
  end loop;

  return v_result;
end;
$$;


revoke all on function public.livira_create_inventories(jsonb) from public, anon, authenticated;
grant execute on function public.livira_create_inventories(jsonb) to service_role;

commit;


-- ============================================================
-- MIGRATION 021: DASHBOARD TITIPAN SYNC
-- ============================================================
-- LIVIRA migration 021
-- Menyinkronkan dashboard agar total inventory aktif selalu merupakan jumlah
-- BTD + BDN + BMMN + Barang Titipan, termasuk metrik per TPP.

begin;

create or replace function public.livira_dashboard_summary()
returns jsonb
language sql
stable
security definer
set search_path = public, pg_temp
as $$
with active_items as (
  select *
  from public.inventory_items
  where is_active = true
    and item_type in ('BTD', 'BDN', 'BMMN', 'TITIPAN')
),
eligible_occupancy as (
  select distinct on (facility_id, unit_key)
    facility_id,
    case
      when upper(load_type) = 'FCL' then
        case upper(container_size)
          when '40' then 2::numeric
          when '40HC' then 2::numeric
          when '45' then 2.25::numeric
          when '45HC' then 2.25::numeric
          else 1::numeric
        end
      else 0::numeric
    end as yard_used,
    case
      when upper(load_type) = 'LCL' and estimated_volume_m3 > 0 then estimated_volume_m3
      else 0::numeric
    end as shed_used
  from (
    select ai.*,
      case
        when trim(ai.physical_unit_id) <> '' then trim(ai.physical_unit_id)
        when upper(ai.load_type) = 'FCL' and trim(ai.container_no) <> ''
          then 'FCL:' || upper(regexp_replace(ai.container_no, '[ .-]', '', 'g'))
        else 'ITEM:' || ai.id::text
      end as unit_key
    from active_items ai
    where ai.at_tpp = true
      and ai.facility_id is not null
      and (trim(ai.physical_unit_id) = '' or ai.occupancy_primary = true)
  ) candidates
  order by facility_id, unit_key, updated_at desc
),
facility_rows as (
  select
    f.id as facility_id,
    f.name as facility_name,
    count(ai.id) filter (where ai.item_type = 'BTD')::int as btd,
    count(ai.id) filter (where ai.item_type = 'BDN')::int as bdn,
    count(ai.id) filter (where ai.item_type = 'BMMN')::int as bmmn,
    count(ai.id) filter (where ai.item_type = 'TITIPAN')::int as titipan,
    count(ai.id) filter (where ai.item_type in ('BTD', 'BDN', 'BMMN', 'TITIPAN'))::int as total,
    f.yard_capacity,
    coalesce((select sum(eo.yard_used) from eligible_occupancy eo where eo.facility_id = f.id), 0) as yard_used,
    f.shed_capacity,
    coalesce((select sum(eo.shed_used) from eligible_occupancy eo where eo.facility_id = f.id), 0) as shed_used,
    f.sort_order
  from public.facilities f
  left join active_items ai on ai.facility_id = f.id
  where f.active = true
  group by f.id, f.name, f.yard_capacity, f.shed_capacity, f.sort_order
),
process_counts as (
  select
    count(*) filter (where is_active and disposition_type = 'lelang')::int as auction_active,
    count(*) filter (where is_active and disposition_type = 'musnah')::int as destruction_active,
    count(*) filter (where is_active and disposition_type = 'hibah')::int as grant_active,
    count(*) filter (
      where not is_active
        and date_trunc('month', updated_at) = date_trunc('month', now())
    )::int as completed_this_month
  from public.dispositions
),
attention as (
  select coalesce(jsonb_agg(to_jsonb(q) - 'search_text' order by q.determination_date asc), '[]'::jsonb) as rows
  from (
    select *
    from active_items
    where coalesce(determination_date, created_at) <= now() - interval '45 days'
    order by determination_date asc nulls first, created_at asc
    limit 5
  ) q
),
recent as (
  select coalesce(jsonb_agg(to_jsonb(q) order by q.created_at desc), '[]'::jsonb) as rows
  from (
    select *
    from public.events
    order by created_at desc
    limit 6
  ) q
),
item_counts as (
  select
    count(*) filter (where item_type = 'BTD')::int as btd_total,
    count(*) filter (where item_type = 'BDN')::int as bdn_total,
    count(*) filter (where item_type = 'BMMN')::int as bmmn_total,
    count(*) filter (where item_type = 'TITIPAN')::int as titipan_total
  from active_items
),
facility_json as (
  select
    coalesce(jsonb_agg(
      jsonb_build_object(
        'facility_id', facility_id,
        'facility_name', facility_name,
        'btd', btd,
        'bdn', bdn,
        'bmmn', bmmn,
        'titipan', titipan,
        'total', total,
        'yard_capacity', yard_capacity,
        'yard_used', yard_used,
        'shed_capacity', shed_capacity,
        'shed_used', shed_used
      ) order by sort_order, facility_name
    ), '[]'::jsonb) as rows,
    coalesce(sum(yard_capacity), 0) as yard_capacity,
    coalesce(sum(yard_used), 0) as yard_used,
    coalesce(sum(shed_capacity), 0) as shed_capacity,
    coalesce(sum(shed_used), 0) as shed_used
  from facility_rows
)
select jsonb_build_object(
  'active_total', ic.btd_total + ic.bdn_total + ic.bmmn_total + ic.titipan_total,
  'btd_total', ic.btd_total,
  'bdn_total', ic.bdn_total,
  'bmmn_total', ic.bmmn_total,
  'titipan_total', ic.titipan_total,
  'active_summary', public.livira_inventory_dashboard_metrics(null),
  'btd_summary', public.livira_inventory_dashboard_metrics('BTD'),
  'bdn_summary', public.livira_inventory_dashboard_metrics('BDN'),
  'bmmn_summary', public.livira_inventory_dashboard_metrics('BMMN'),
  'titipan_summary', public.livira_inventory_dashboard_metrics('TITIPAN'),
  'auction_active', pc.auction_active,
  'destruction_active', pc.destruction_active,
  'grant_active', pc.grant_active,
  'completed_this_month', pc.completed_this_month,
  'occupancy', jsonb_build_object(
    'yard_capacity', fj.yard_capacity,
    'yard_used', fj.yard_used,
    'shed_capacity', fj.shed_capacity,
    'shed_used', fj.shed_used
  ),
  'facility_breakdown', fj.rows,
  'recent_events', r.rows,
  'attention_items', a.rows
)
from item_counts ic
cross join process_counts pc
cross join facility_json fj
cross join recent r
cross join attention a;
$$;

revoke all on function public.livira_dashboard_summary() from public, anon, authenticated;
grant execute on function public.livira_dashboard_summary() to service_role;

commit;


-- ============================================================================
-- BEGIN MIGRATION: 028_dashboard_scope_pindah_bongkar_kontainer.sql
-- ============================================================================
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
      v_clone.status_label := 'Bongkar/Muat Kontainer';
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

-- END MIGRATION: 028_dashboard_scope_pindah_bongkar_kontainer.sql

-- ============================================================================
-- BEGIN MIGRATION: 029_livira_rebrand_dashboard_scope_container_target.sql
-- ============================================================================

-- Fresh setup already creates objects with prefix livira_. The final migration
-- marker ensures the private document bucket and application identity exist.
insert into storage.buckets (id, name, public, file_size_limit, allowed_mime_types)
values (
  'livira-documents',
  'livira-documents',
  false,
  8388608,
  array['application/pdf','image/jpeg','image/png','image/webp','image/gif']
)
on conflict (id) do update
set
  name = excluded.name,
  public = false,
  file_size_limit = excluded.file_size_limit,
  allowed_mime_types = excluded.allowed_mime_types;

comment on schema public is
  'LIVIRA — Layanan Inventori, Verifikasi, Integrasi, Rekonsiliasi, dan Analitik';

-- ============================================================================
-- END MIGRATION: 029_livira_rebrand_dashboard_scope_container_target.sql
-- ============================================================================

-- ============================================================================
-- BEGIN HOTFIX: 030_fix_rebrand_function_body_and_postgrest_cache.sql
-- ============================================================================
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
-- ============================================================================
-- END HOTFIX: 030_fix_rebrand_function_body_and_postgrest_cache.sql
-- ============================================================================

-- ============================================================================
-- BEGIN MIGRATION: 031_dashboard_office_scope_granular_inventory_access.sql
-- ============================================================================
-- LIVIRA revision 031
-- 1. Migrasi permission Kelola Inventory lama ke permission granular.
-- 2. Izinkan bongkar/muat pada semua inventory aktif kecuali barang yang sudah keluar.
--    Barang yang sedang/sudah dalam proses penyelesaian hanya boleh dipindahkan
--    ke satu tujuan agar relasi proses dan nilai barang tetap utuh.

begin;

with migrated_permissions as (
  select
    r.id,
    coalesce(jsonb_agg(p.permission order by p.permission), '[]'::jsonb) as permissions
  from public.app_roles r
  cross join lateral (
    select distinct permission
    from (
      select value as permission
      from jsonb_array_elements_text(coalesce(r.permissions, '[]'::jsonb)) as existing(value)
      where value <> 'inventory.manage'

      union all select 'inventory.action.pemindahan' where r.permissions ? 'inventory.manage'
      union all select 'inventory.action.bongkar_muat' where r.permissions ? 'inventory.manage'
      union all select 'inventory.action.pemberitahuan' where r.permissions ? 'inventory.manage'
      union all select 'inventory.action.pencacahan' where r.permissions ? 'inventory.manage'
      union all select 'inventory.action.request_penelitian_pfpd' where r.permissions ? 'inventory.manage'
      union all select 'inventory.action.penelitian_pfpd' where r.permissions ? 'inventory.manage'
      union all select 'inventory.action.penetapan_bmmn' where r.permissions ? 'inventory.manage'
      union all select 'inventory.action.usulan_peruntukan_bmmn' where r.permissions ? 'inventory.manage'
      union all select 'inventory.action.persetujuan_peruntukan_bmmn' where r.permissions ? 'inventory.manage'
      union all select 'inventory.action.pengeluaran_barang' where r.permissions ? 'inventory.manage'

      union all select 'inventory.create.btd'
        where r.permissions ? 'inventory.manage' and r.permissions ? 'inventory.type.btd'
      union all select 'inventory.create.bdn'
        where r.permissions ? 'inventory.manage' and r.permissions ? 'inventory.type.bdn'
      union all select 'inventory.create.titipan'
        where r.permissions ? 'inventory.manage' and r.permissions ? 'inventory.type.titipan'
    ) expanded
  ) p
  where r.permissions ? 'inventory.manage'
  group by r.id
)
update public.app_roles r
set permissions = m.permissions,
    updated_at = now()
from migrated_permissions m
where r.id = m.id
  and r.permissions is distinct from m.permissions;

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
  if coalesce(v_source.quantity, 0) <= 0 then
    raise exception 'invalid transition: inventory quantity must be positive' using errcode = 'P0001';
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
  if (v_source.current_disposition is not null
      or v_source.status_code in ('laku', 'alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima'))
     and v_count > 1 then
    raise exception 'invalid transition: processed inventory may only be relocated to one destination' using errcode = 'P0001';
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
          status_code = case
            when v_source.current_disposition is null
             and v_source.status_code not in ('laku', 'alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima')
            then 'pindah_bongkar_kontainer'
            else v_source.status_code
          end,
          status_label = case
            when v_source.current_disposition is null
             and v_source.status_code not in ('laku', 'alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima')
            then 'Bongkar/Muat Kontainer'
            else v_source.status_label
          end,
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
      v_clone.status_label := 'Bongkar/Muat Kontainer';
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
    'Bongkar/Muat Kontainer',
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
  'Memindahkan penempatan fisik inventory aktif. Barang yang sedang atau sudah berproses tetap dapat dipindahkan satu-ke-satu tanpa mengubah status proses; inventory biasa tetap dapat dibagi ke beberapa tujuan.';

commit;

notify pgrst, 'reload schema';

select
  to_regprocedure('public.livira_relocate_inventory_load(uuid,timestamptz,jsonb,jsonb)') is not null
    as livira_relocate_inventory_load_ready,
  count(*) filter (where permissions ? 'inventory.manage') as roles_with_legacy_inventory_manage
from public.app_roles;
-- ============================================================================
-- END MIGRATION: 031_dashboard_office_scope_granular_inventory_access.sql
-- ============================================================================

-- ============================================================================
-- BEGIN MIGRATION: 032_bongkar_muat_preserve_inventory_status.sql
-- LIVIRA revision 032
-- Bongkar/muat kontainer hanya mengubah penempatan fisik dan tidak pernah
-- menjadi status inventory. Migration ini juga memulihkan data lama yang
-- sempat berstatus Bongkar/Muat Kontainer.

begin;

-- Pulihkan inventory lama yang terlanjur memiliki status action bongkar/muat.
-- Status diambil dari event terakhir sebelum event bongkar/muat. Jika tidak ada,
-- gunakan status awal berdasarkan jenis dan lokasi barang.
with previous_status as (
  select
    i.id,
    coalesce(prev.code,
      case
        when i.item_type = 'TITIPAN' then 'barang_titipan_aktif'
        when i.at_tpp then 'ditetapkan'
        else 'masih_di_tps'
      end
    ) as status_code,
    coalesce(prev.label,
      case
        when i.item_type = 'TITIPAN' then 'Barang titipan aktif'
        when i.at_tpp then 'Ditetapkan sebagai ' || i.item_type
        else 'Masih di TPS'
      end
    ) as status_label
  from public.inventory_items i
  left join lateral (
    select e.code, e.label
    from public.events e
    where e.inventory_id = i.id
      and e.code <> 'pindah_bongkar_kontainer'
    order by e.created_at desc, e.id desc
    limit 1
  ) prev on true
  where i.status_code = 'pindah_bongkar_kontainer'
)
update public.inventory_items i
set status_code = p.status_code,
    status_label = p.status_label,
    updated_at = now()
from previous_status p
where i.id = p.id;

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
  if coalesce(v_source.quantity, 0) <= 0 then
    raise exception 'invalid transition: inventory quantity must be positive' using errcode = 'P0001';
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
  if (v_source.current_disposition is not null
      or v_source.status_code in ('laku', 'alokasi_hasil_lelang', 'ba_musnah', 'ba_serah_terima'))
     and v_count > 1 then
    raise exception 'invalid transition: processed inventory may only be relocated to one destination' using errcode = 'P0001';
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
          -- Bongkar/muat hanya mengubah penempatan fisik. Status proses tetap.
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
      v_clone.status_code := v_source.status_code;
      v_clone.status_label := v_source.status_label;
      v_clone.current_disposition := v_source.current_disposition;
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
    'Bongkar/Muat Kontainer',
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
  'Memindahkan penempatan fisik inventory aktif tanpa mengubah status barang. Barang yang sedang atau sudah berproses tetap dapat dipindahkan satu-ke-satu; inventory biasa tetap dapat dibagi ke beberapa tujuan.';

commit;

notify pgrst, 'reload schema';
-- ============================================================================
-- END MIGRATION: 032_bongkar_muat_preserve_inventory_status.sql
-- ============================================================================

-- ============================================================================
-- VERIFIKASI AKHIR SETUP REVISI 033
-- Seluruh kolom *_ready harus bernilai true dan inventory_rows harus 0.
-- ============================================================================
select
  to_regclass('public.facilities') is not null                    as facilities_ready,
  to_regclass('public.inventory_items') is not null               as inventory_items_ready,
  to_regclass('public.dispositions') is not null                  as dispositions_ready,
  to_regclass('public.events') is not null                        as events_ready,
  to_regclass('public.reconciliations') is not null               as reconciliations_ready,
  to_regclass('public.uploaded_documents') is not null            as uploaded_documents_ready,
  to_regclass('public.inventory_deletion_audit') is not null      as deletion_audit_ready,
  to_regclass('public.audit_logs') is not null                    as audit_logs_ready,
  to_regclass('public.app_users') is not null                     as app_users_ready,
  to_regclass('public.app_roles') is not null                     as app_roles_ready,
  to_regclass('public.app_parameters') is not null                as app_parameters_ready,
  to_regprocedure('public.livira_dashboard_summary()') is not null
                                                                  as dashboard_rpc_ready,
  to_regprocedure('public.livira_inventory_dashboard_metrics(text)') is not null
                                                                  as dashboard_metrics_rpc_ready,
  to_regprocedure('public.livira_relocate_inventory_load(uuid,timestamptz,jsonb,jsonb)') is not null
                                                                  as bongkar_muat_rpc_ready,
  (
    select count(*)
    from pg_constraint c
    join pg_class t on t.oid = c.conrelid
    join pg_namespace n on n.oid = t.relnamespace
    where n.nspname = 'public'
      and t.relname = 'app_users'
      and c.contype = 'f'
      and c.confdeltype = 'c'
  ) > 0                                                           as auth_user_delete_cascade_ready,
  (select count(*) from public.inventory_items)                   as inventory_rows,
  (select count(*) from public.facilities)                        as facility_master_rows,
  (select count(*) from public.app_roles where active = true)     as active_role_rows,
  (select count(*) from public.app_parameters where active = true) as active_parameter_rows;
