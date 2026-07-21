# Revisi 024 — Ekspor Excel untuk Seluruh Laporan

Menu **Pelaporan** sekarang menyediakan dua pilihan ekspor untuk setiap laporan non-performa:

- **CSV UTF-8** untuk kebutuhan pertukaran data sederhana.
- **Excel `.xlsx`** untuk pengolahan dan penyajian lanjutan di Microsoft Excel atau aplikasi spreadsheet yang kompatibel.

Cakupan ekspor Excel meliputi:

1. Laporan kustom sesuai seluruh kombinasi filter.
2. Barang aktif per TPP.
3. BTD/BDN berumur sekurangnya 60 hari.
4. Potensi barang siap lelang.
5. Barang aktif yang masih berada di TPS.
6. BMMN yang menunggu peruntukan.
7. Riwayat barang selesai.
8. Laporan BTD lengkap.
9. Rekap rekonsiliasi fisik.
10. Rekap perubahan data barang.

CSV dan Excel menggunakan sumber data, susunan kolom, cakupan role, dan filter yang sama. Ekspor selalu mencakup seluruh data yang sesuai filter, bukan hanya halaman tabel yang sedang dibuka.

File Excel memiliki:

- judul laporan dan waktu pembuatan;
- header tabel yang dibedakan secara visual;
- pembekuan baris header;
- filter otomatis pada seluruh kolom;
- lebar kolom yang disesuaikan;
- pembungkusan teks untuk uraian panjang; dan
- format numerik untuk jumlah, umur, serta nilai rupiah.

Endpoint lama CSV tetap tersedia. Endpoint Excel utama menggunakan `/pelaporan.xlsx`. Endpoint `/pelaporan.xls` tetap disediakan sebagai kompatibilitas format Excel lama, tetapi tombol antarmuka menggunakan `.xlsx` sebagai format utama.

Revisi ini tidak mengubah struktur database dan tidak memerlukan migration SQL baru.
