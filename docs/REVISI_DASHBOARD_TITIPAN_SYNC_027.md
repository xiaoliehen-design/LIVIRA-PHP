# Revisi 027: Sinkronisasi Dashboard Barang Titipan

Revisi ini memperbaiki dashboard agar angka **Total inventory aktif** selalu merupakan penjumlahan empat kelompok inventory aktif:

- BTD
- BDN
- BMMN
- Barang Titipan

Migration `021_dashboard_titipan_sync.sql` memperbarui RPC `livira_dashboard_summary()` agar mengembalikan:

- `titipan_total`
- `titipan_summary` untuk jumlah dokumen, FCL, dan LCL Barang Titipan
- kolom `titipan` pada rincian per TPP
- `active_total` yang dihitung langsung dari BTD + BDN + BMMN + TITIPAN

Backend juga memiliki pemeriksaan kompatibilitas. Jika database masih memakai RPC lama dan jumlah kartu tidak sama dengan total inventory, backend mengambil data Barang Titipan aktif untuk memperbaiki angka dashboard. Migration 021 tetap harus dijalankan agar dashboard menggunakan satu RPC yang efisien tanpa query kompatibilitas tambahan.

## Database yang sudah berjalan

Jalankan satu kali:

```text
migrations/021_dashboard_titipan_sync.sql
```

Migration ini tidak menghapus dan tidak mengubah data barang.

## Database baru dan kosong

Jalankan hanya:

```text
migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql
```
