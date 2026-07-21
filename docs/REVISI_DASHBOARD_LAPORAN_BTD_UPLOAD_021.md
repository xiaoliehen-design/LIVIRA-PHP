# Revisi Dashboard, Laporan BTD, Pencacahan, dan Upload Multi-Barang

## Ringkasan perubahan

1. Setiap kartu inventory pada Dashboard menampilkan jumlah dokumen, FCL, dan LCL. Nomor dokumen, nomor kontainer FCL, dan identitas LCL dideduplikasi agar beberapa uraian barang tidak menggandakan angka.
2. Seluruh informasi paginasi yang sebelumnya menampilkan “Halaman X dari X” kini juga menampilkan total item.
3. Preset **Laporan BTD** menampilkan nomor BTD, tanggal BTD, daftar kontainer, serta uraian dan jumlah barang yang dikelompokkan per kontainer. Ekspor CSV dan Excel memakai struktur data yang sama.
4. Action Pencacahan memiliki isian opsional **Detail jumlah barang** untuk setiap hasil pencacahan.
5. Nama menu **Penetapan BTD** diubah menjadi **Pencatatan BTD**.
6. Pencatatan BTD mewajibkan **Nomor BL**; revisi 023 menambahkan **Tanggal BL** sebagai isian wajib dan kelompok form ditampilkan sebagai “BL, manifest, muatan, dan TPS asal”.
7. Serialisasi jumlah barang pada form manual diperbaiki sehingga JSON FCL/LCL dapat divalidasi server setelah seluruh kolom diisi.
8. Upload Excel menerima beberapa baris dengan nomor kontainer yang sama apabila baris-baris tersebut merupakan beberapa identitas barang dalam satu dokumen dan metadata kontainernya konsisten.

## Aturan deduplikasi

- **Dokumen:** jenis inventory + nomor dokumen + tanggal dokumen.
- **FCL:** nomor kontainer yang telah dinormalisasi.
- **LCL:** `physical_unit_id`; bila data lama belum memilikinya, sistem memakai identitas dokumen sebagai fallback.
- **Upload FCL:** baris pertama menjadi `occupancy_primary=true`; baris berikutnya untuk kontainer yang sama menjadi `false`.

## Database

Database yang sudah mencapai migration 018 cukup menjalankan:

```text
migrations/019_btd_dashboard_report_upload_fixes.sql
```

Database Supabase baru dan kosong cukup menjalankan:

```text
migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql
```

Jangan menjalankan file setup penuh pada database operasional yang telah berisi data.

## Pengujian

Pengujian otomatis mencakup deduplikasi dashboard, pengelompokan Laporan BTD, validasi template BTD, dukungan dua barang dalam satu kontainer, dan penolakan nomor kontainer sama pada dokumen yang berbeda.
