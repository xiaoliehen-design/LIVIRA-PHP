# Revisi 025 — Filter Laporan BTD dan Perbaikan File Excel

Preset **Laporan BTD** sekarang menyediakan filter khusus yang tetap mempertahankan format rekap satu baris per dokumen BTD.

Filter yang tersedia:

1. tanggal BTD dari tanggal tertentu;
2. tanggal BTD sampai tanggal tertentu;
3. status inventory: semua, aktif saja, atau selesai saja;
4. status barang/tahapan proses;
5. lokasi barang: semua lokasi, masih di TPS, atau sudah di TPP; dan
6. TPP tertentu.

Filter diterapkan secara konsisten pada tabel, pagination, ekspor CSV, dan ekspor Excel. Jika tanggal awal lebih besar daripada tanggal akhir, sistem menormalkan urutannya secara otomatis. Default Laporan BTD tetap mencakup inventory aktif dan selesai.

## Perbaikan ekspor Excel

Struktur internal workbook `.xlsx` diperbaiki agar mengikuti urutan elemen Open XML yang diterima Microsoft Excel. Perbaikan mencakup:

- metadata dimensi worksheet;
- informasi tampilan workbook;
- metadata baris dan panel beku;
- urutan `autoFilter`, area sel gabungan, margin halaman, dan konfigurasi halaman;
- pemeriksaan XML serta pengujian struktur workbook otomatis.

Perbaikan ini mencegah pesan Microsoft Excel **“We found a problem with some content…”** pada file laporan yang diekspor. Sumber data, isi kolom, filter, dan endpoint ekspor tidak berubah.

Revisi ini tidak mengubah struktur database dan tidak memerlukan migration SQL baru.
