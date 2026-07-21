# Revisi 031 — Cakupan Kantor, Hak Akses Inventory, dan Bongkar/Muat

## Perubahan dashboard

Dropdown **Cakupan inventory** memiliki urutan:

1. Seluruh cakupan Kantor Tanjung Priok — seluruh inventory aktif, baik masih di TPS maupun sudah berada di TPP.
2. Masih di TPS.
3. Seluruh TPP.
4. Masing-masing TPP aktif.

Filter tersebut hanya mengubah kartu Total Inventory Aktif, BTD, BDN, BMMN, Barang Titipan, ringkasan dokumen/FCL/LCL, dan daftar perhatian. Tabel **Detail per TPP** selalu mengambil seluruh barang aktif yang benar-benar berada di tiap TPP sehingga angkanya tidak berubah saat filter KPI diganti.

## Hak akses inventory granular

Permission input awal:

- Pencatatan BTD.
- Penetapan BDN.
- Pemasukan barang titipan.

Permission action:

- Pemindahan.
- Bongkar/Muat Kontainer.
- Pemberitahuan.
- Pencacahan.
- Request Penelitian PFPD.
- Penelitian PFPD.
- Penetapan BMMN.
- Usulan Peruntukan BMMN.
- Persetujuan Peruntukan BMMN.
- Pengeluaran Barang.

Menu dan tombol hanya ditampilkan apabila role mempunyai permission terkait. Backend juga memeriksa permission yang tepat sehingga pembatasan tidak hanya bergantung pada tampilan. Permission jenis barang BTD/BDN/BMMN/Titipan tetap menjadi pembatas cakupan data.

Migration 031 otomatis mengubah role yang masih memiliki `inventory.manage`. Permission action lama dipetakan ke seluruh action granular, sedangkan permission input awal hanya ditambahkan sesuai cakupan jenis barang role tersebut. Trigger revokasi sesi akan meminta pengguna terkait login kembali.

## Bongkar/Muat pada barang yang sedang berproses

Action Bongkar/Muat tersedia untuk semua barang yang masih aktif. Barang yang sedang atau sudah melalui proses lelang, pemusnahan, hibah, atau PSP tetap dapat dipindahkan secara fisik. Status dan `current_disposition` proses dipertahankan.

Untuk menjaga hubungan proses, timeline, dan nilai barang, satu uraian yang sudah terikat proses hanya dapat dialokasikan ke **satu tujuan** dalam satu action. Inventory yang belum terikat proses tetap dapat dibagi ke beberapa kontainer atau lot LCL dengan konservasi kuantitas dan nilai. Barang yang sudah dikeluarkan (`is_active = false`) tidak dapat diproses.

## Urutan implementasi database operasional

1. Pastikan migration 029 dan hotfix 030 sudah berhasil.
2. Jalankan `migrations/031_dashboard_office_scope_granular_inventory_access.sql` di Supabase SQL Editor.
3. Deploy source revisi 031.
4. Login ulang untuk akun yang role-nya dimigrasikan.

Migration 031 tidak menghapus atau mereset data barang.
