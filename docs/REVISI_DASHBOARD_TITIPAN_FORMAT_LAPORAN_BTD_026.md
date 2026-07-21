# Revisi 026 — Dashboard Barang Titipan dan Format Laporan BTD

## Perubahan dashboard

- Menambahkan kartu KPI Barang Titipan pada dashboard.
- Total barang titipan, jumlah dokumen, FCL, dan LCL dihitung dari inventory aktif dengan deduplikasi yang sama seperti jenis inventory lain.
- Detail per TPP menampilkan kolom Barang Titipan agar jumlah BTD, BDN, BMMN, Titipan, dan Total dapat direkonsiliasi secara langsung.

## Perubahan Laporan BTD

- Menambahkan kolom Jumlah Kontainer setelah kolom Kontainer/LCL.
- Jumlah kontainer dihitung dari nomor kontainer FCL unik dalam satu dokumen BTD. Baris LCL tidak menambah jumlah kontainer.
- Kolom Kontainer/LCL tidak lagi menggunakan awalan “Kontainer 1”, “Kontainer 2”, dan seterusnya. Nomor kontainer langsung dipisahkan dengan titik koma.
- Kolom uraian langsung diawali nomor kontainer, misalnya `ABCD1234567(Mesin [Barang Umum]: 5 Piece)`.
- Format tabel, CSV, XLSX, dan XLS dibuat konsisten.

Revisi ini tidak memerlukan migration SQL baru.
