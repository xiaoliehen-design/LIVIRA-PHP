# Revisi 023 — Tanggal BL pada Pencatatan BTD

## Perubahan

- Menambahkan isian **Tanggal BL** tepat di samping **Nomor BL** pada form Pencatatan BTD.
- Nomor BL dan Tanggal BL wajib diisi hanya untuk jenis inventory BTD.
- Menambahkan kolom `bl_date` pada database melalui migration 020.
- Menambahkan Tanggal BL pada detail inventory, Perubahan Data Barang, preset Laporan BTD, dan ekspor CSV maupun Excel.
- Laporan BTD bertambah dari 17 menjadi 18 kolom.
- Template upload Excel Pencatatan BTD sekarang memiliki kolom **Tanggal BL \*** dan validasi format `dd/mm/yyyy`.
- Untuk beberapa uraian barang dalam kontainer atau dokumen yang sama, Nomor BL dan Tanggal BL harus konsisten.

## Database existing

Jalankan satu kali:

```sql
-- isi file migrations/020_btd_bl_date.sql
```

Migration tidak menghapus atau mereset data lama. Data BTD historis yang belum memiliki tanggal BL tetap boleh kosong, sedangkan pencatatan baru diwajibkan oleh aplikasi.

## Database baru kosong

Jalankan hanya:

```text
migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql
```
