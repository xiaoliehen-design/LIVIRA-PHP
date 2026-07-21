# Revisi Barang Titipan, Rekonsiliasi, dan Alur Lelang

## Implementasi utama

1. Inventory memiliki tab **Barang Titipan** serta tombol **Pemasukan barang titipan kantor/unit lain** di antara tombol History dan Action.
2. Pemasukan barang titipan mencatat nomor/tanggal dokumen dasar, kategori BTD/BDN/BMMN/Tidak Teridentifikasi, kantor/unit penitip, manifest, muatan, kontainer/LCL, data barang, lokasi, dan shipper/consignee tanpa TPS asal.
3. Status lokasi menampilkan langsung nama TPS, TPP, atau kantor/unit penitip.
4. Jenis pengeluaran divalidasi berdasarkan jenis inventory. Barang titipan hanya dapat memakai **PENGELUARAN BARANG TITIPAN**.
5. Preset **Jumlah penyelesaian** dan **Jumlah konversi ke BMMN** dihapus dari pelaporan.
6. Dashboard Lelang, Musnah, dan Hibah/PSP memisahkan **proses dimulai tahun ini** dan **selesai tahun ini**.
7. Menu **Rekonsiliasi** dapat mengeluarkan barang yang tercatat tetapi tidak ditemukan atau menambahkan barang fisik yang belum tercatat beserta status sebenarnya dan catatan audit.
8. Preset serta ekspor CSV dan Excel **Laporan rekonsiliasi** tersedia pada menu Pelaporan.
9. Role & Hak Akses memiliki izin **Akses barang titipan**, **Lihat rekonsiliasi**, dan **Kelola rekonsiliasi**.
10. Action **Selesai Lelang** memilih satu ND penjadwalan. Status laku/tidak laku dan harga jual diisi per barang, sedangkan nomor serta tanggal risalah diterapkan ke seluruh bundle. Tidak ada komponen biaya lelang.
11. Nilai utama pada menu Lelang dan pengurutan nilai menggunakan HTL.
12. Barang dengan KEP Musnah dapat dikeluarkan dari inventory aktif, tetapi proses tetap tampil pada menu Musnah sampai BA Musnah selesai.

## Cara upgrade

1. Gunakan source code versi ini.
2. Pastikan migration `001` sampai `012` sudah pernah dijalankan sesuai urutan.
3. Jalankan isi file berikut pada Supabase SQL Editor:

   `migrations/013_titipan_rekonsiliasi_lelang_dashboard.sql`

4. Deploy ulang aplikasi.
5. Buka **Admin → Role & Hak Akses** dan sesuaikan akses Barang Titipan/Rekonsiliasi untuk role yang membutuhkan.

Migration `013` aman dijalankan ulang. Migration menambahkan kolom dan tabel baru, memperbarui constraint, menormalkan nama lokasi, melakukan backfill ND penjadwalan lama, dan memperbarui parameter jenis pengeluaran.
