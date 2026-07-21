-- =============================================================================
-- LIVIRA — RESET SELURUH DATA BARANG (KOMPATIBEL SETUP 001 s.d. 029)
-- =============================================================================
-- PERINGATAN:
--   1. Operasi ini permanen setelah COMMIT dan tidak dapat di-undo.
--   2. Jalankan hanya pada database LIVIRA yang skemanya sudah terpasang.
--   3. Script menghapus seluruh inventory aktif maupun selesai/history untuk:
--        - BTD
--        - BDN
--        - BMMN
--        - Barang Titipan
--      beserta proses, timeline, rekonsiliasi, perubahan data, dokumen, snapshot
--      penghapusan, dan audit operasional/laporan yang terkait dengan barang.
--   4. Script tetap mempertahankan:
--        - akun Supabase Auth;
--        - app_users, app_roles, dan pengaturan akses;
--        - app_parameters;
--        - master TPP/facilities dan nilai kapasitas YOR/SOR.
--      Hanya nilai pemakaian YOR/SOR yang dikembalikan menjadi 0.
--
-- FILE SUPABASE STORAGE:
--   Metadata uploaded_documents akan dihapus oleh SQL ini. File fisik pada bucket
--   "livira-documents" sebaiknya dihapus melalui Supabase Storage Dashboard/API,
--   bukan dengan DELETE langsung ke storage.objects, agar tidak meninggalkan file
--   fisik yatim atau metadata yang tidak sinkron.
-- =============================================================================

begin;

-- Jangan menunggu tanpa batas jika aplikasi masih menulis data saat reset.
set local lock_timeout = '15s';

-- Pastikan script dijalankan pada skema LIVIRA yang benar.
do $$
begin
  if to_regclass('public.inventory_items') is null
     or to_regclass('public.dispositions') is null
     or to_regclass('public.events') is null
     or to_regclass('public.reconciliations') is null
     or to_regclass('public.uploaded_documents') is null
     or to_regclass('public.inventory_deletion_audit') is null
     or to_regclass('public.audit_logs') is null
     or to_regclass('public.facilities') is null then
    raise exception 'Skema LIVIRA belum lengkap. Jalankan SQL setup database terlebih dahulu.';
  end if;
end;
$$;

-- Mencegah penambahan/perubahan data barang selama proses reset berlangsung.
lock table public.inventory_items in access exclusive mode;
lock table public.dispositions in access exclusive mode;
lock table public.events in access exclusive mode;
lock table public.reconciliations in access exclusive mode;
lock table public.uploaded_documents in access exclusive mode;
lock table public.inventory_deletion_audit in access exclusive mode;
lock table public.audit_logs in access exclusive mode;
lock table public.facilities in row exclusive mode;

-- Simpan jumlah awal untuk hasil verifikasi yang ditampilkan setelah reset.
drop table if exists livira_reset_before;
create temporary table livira_reset_before (
  inventory_count bigint,
  disposition_count bigint,
  event_count bigint,
  reconciliation_count bigint,
  document_count bigint,
  deletion_audit_count bigint,
  operational_audit_count bigint
) on commit preserve rows;

insert into livira_reset_before
select
  (select count(*) from public.inventory_items),
  (select count(*) from public.dispositions),
  (select count(*) from public.events),
  (select count(*) from public.reconciliations),
  (select count(*) from public.uploaded_documents),
  (select count(*) from public.inventory_deletion_audit),
  (
    select count(*)
    from public.audit_logs
    where entity_type in (
        'inventory',
        'inventory_batch',
        'disposition',
        'disposition_batch',
        'research_request',
        'reconciliation',
        'document',
        'auction_schedule',
        'report'
      )
       or action like 'inventory.%'
       or action like 'process.%'
       or action like 'reconciliation.%'
       or action like 'document.%'
       or action like 'report.%'
  );

-- Urutan penghapusan mengikuti relasi foreign key.
-- Rekonsiliasi dapat merujuk inventory dan dokumen.
delete from public.reconciliations;

-- Inventory direferensikan dispositions dengan ON DELETE RESTRICT.
-- Events yang terkait disposition ikut terhapus melalui ON DELETE CASCADE.
delete from public.dispositions;

-- Hapus sisa timeline/event yang tidak ikut terhapus pada langkah sebelumnya.
delete from public.events;

-- Hapus seluruh barang aktif dan history dari semua kategori inventory.
delete from public.inventory_items;

-- Setelah event dan rekonsiliasi hilang, metadata dokumen tidak lagi direferensikan.
delete from public.uploaded_documents;

-- Hapus snapshot barang yang sebelumnya dihapus oleh administrator.
delete from public.inventory_deletion_audit;

-- Hapus audit operasional dan ekspor laporan yang bersumber dari data barang.
-- Audit autentikasi, pengguna, role, parameter, dan perubahan kapasitas dipertahankan.
delete from public.audit_logs
where entity_type in (
    'inventory',
    'inventory_batch',
    'disposition',
    'disposition_batch',
    'research_request',
    'reconciliation',
    'document',
    'auction_schedule',
    'report'
  )
   or action like 'inventory.%'
   or action like 'process.%'
   or action like 'reconciliation.%'
   or action like 'document.%'
   or action like 'report.%';

-- Pertahankan kapasitas fasilitas, tetapi kosongkan pemakaiannya.
update public.facilities
set yard_used = 0,
    shed_used = 0;

-- Gagalkan dan rollback otomatis apabila masih ada data operasional tersisa.
do $$
begin
  if exists (select 1 from public.inventory_items)
     or exists (select 1 from public.dispositions)
     or exists (select 1 from public.events)
     or exists (select 1 from public.reconciliations)
     or exists (select 1 from public.uploaded_documents)
     or exists (select 1 from public.inventory_deletion_audit) then
    raise exception 'Reset tidak lengkap. Transaksi dibatalkan agar database tidak berada pada kondisi parsial.';
  end if;
end;
$$;

commit;

-- =============================================================================
-- HASIL VERIFIKASI
-- Nilai kolom *_after harus 0. Kapasitas YOR/SOR tetap dipertahankan.
-- =============================================================================
select
  b.inventory_count          as inventory_before,
  b.disposition_count        as disposition_before,
  b.event_count              as event_before,
  b.reconciliation_count     as reconciliation_before,
  b.document_count           as document_before,
  b.deletion_audit_count     as deletion_audit_before,
  b.operational_audit_count  as operational_audit_before,
  (select count(*) from public.inventory_items)          as inventory_after,
  (select count(*) from public.dispositions)             as disposition_after,
  (select count(*) from public.events)                   as event_after,
  (select count(*) from public.reconciliations)          as reconciliation_after,
  (select count(*) from public.uploaded_documents)       as document_after,
  (select count(*) from public.inventory_deletion_audit) as deletion_audit_after,
  (select coalesce(sum(yard_used), 0) from public.facilities) as total_yor_used_after,
  (select coalesce(sum(shed_used), 0) from public.facilities) as total_sor_used_after
from livira_reset_before b;
