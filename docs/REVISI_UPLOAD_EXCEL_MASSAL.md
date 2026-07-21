# Revisi Upload Excel Massal Inventory

Revisi ini menambahkan metode input massal pada popup:

- Pencatatan BTD;
- Penetapan BDN; dan
- Pemasukan Barang Titipan kantor/unit lain.

## Cara menggunakan

1. Buka menu **Inventory**.
2. Klik **Pencatatan BTD**, **Penetapan BDN**, atau **Pemasukan barang titipan kantor/unit lain**.
3. Pilih tab **Upload Excel**.
4. Unduh template yang tampil pada popup. Template berbeda untuk setiap jenis pemasukan.
5. Hapus atau timpa contoh pada baris kedua, lalu isi data mulai dari baris kedua dan seterusnya.
6. Pilih file `.xlsx`, kemudian klik **Upload dan simpan ke inventory**.

## Aturan file

- Maksimal 1.000 baris data dan ukuran file 6 MB.
- Satu baris mewakili satu identitas/uraian barang. Satu kontainer FCL dapat diulang pada beberapa baris untuk mencatat beberapa uraian atau jenis barang.
- Untuk beberapa kontainer dalam satu nomor dokumen, ulangi data dokumen pada setiap baris. Nomor kontainer boleh sama apabila baris tersebut merupakan barang lain dalam kontainer yang sama; ukuran, BL, manifest, TPS/TPP, dan status lokasinya harus konsisten.
- FCL wajib mengisi nomor dan ukuran kontainer. Hanya baris pertama dari kontainer yang sama yang dihitung sebagai unit fisik/YOR.
- LCL wajib mengisi perkiraan volume dalam m³. Beberapa uraian LCL pada dokumen yang sama berbagi satu identitas unit fisik.
- Nomor BL dan Tanggal BL wajib diisi pada template Pencatatan BTD.
- Tanggal dapat diisi dengan format `dd/mm/yyyy`.
- Nilai kategori, TPS, TPP, jenis barang, satuan, dan jenis muatan harus sesuai dengan parameter aktif aplikasi.
- Seluruh file divalidasi terlebih dahulu. Jika satu baris salah, seluruh upload dibatalkan sehingga tidak terjadi penyimpanan sebagian.

## Template yang disertakan

- `internal/web/static/templates/template_upload_btd.xlsx`
- `internal/web/static/templates/template_upload_bdn.xlsx`
- `internal/web/static/templates/template_upload_barang_titipan.xlsx`

Masing-masing template memiliki sheet data utama, sheet referensi pilihan, sheet petunjuk, dan contoh isian pada baris kedua.

## Database

Versi awal upload massal diperkenalkan bersama migration 013. Revisi nomor BL, deduplikasi unit fisik, dan dukungan beberapa barang dalam satu kontainer menggunakan:

```text
migrations/019_btd_dashboard_report_upload_fixes.sql
```

Untuk database baru yang masih kosong, jalankan hanya `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`.
