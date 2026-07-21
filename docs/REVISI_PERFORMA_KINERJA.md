# Revisi Performa Kinerja

## Dashboard

Kartu **Selesai bulan ini** diganti menjadi **Performa kinerja**. Saat diklik, kartu membuka popup dengan periode default satu tahun kalender berjalan dan filter tanggal dari–sampai.

Metrik yang ditampilkan:

1. Lelang selesai: dari penetapan awal BTD/BDN sampai Risalah Lelang.
2. Pemusnahan selesai: dari penetapan awal BTD/BDN sampai BA Musnah.
3. Hibah/PSP selesai: dari penetapan awal BTD/BDN sampai BA Serah Terima.
4. Pencacahan selesai: dari penetapan awal sampai BA Cacah.
5. Penilaian PFPD selesai: dari request penelitian PFPD sampai dokumen penilaian.
6. Konversi BMMN: dari penetapan awal BTD/BDN sampai KEP BMMN.

Untuk barang yang telah menjadi BMMN, tanggal awal tetap menggunakan `origin_document_date`, bukan tanggal KEP BMMN. Satu dokumen penyelesaian yang mencakup banyak barang dihitung sebagai satu penyelesaian agar multi-uraian tidak menggandakan angka kinerja.

## Pelaporan

Preset **Performa kinerja** tersedia pada menu Pelaporan. Pengguna dapat mengatur rentang tanggal dan mengunduh file `.xlsx` yang berisi:

- sheet **Ringkasan**: jumlah selesai dan waktu rata-rata per kategori;
- sheet **Rincian**: dokumen selesai, dokumen awal/request, tanggal, durasi, dan jumlah barang dalam dokumen.

Rentang tanggal menggunakan tanggal dokumen penyelesaian. Jika dokumen lama tidak mempunyai tanggal dokumen, sistem memakai timestamp event sebagai fallback.

## Database

Tidak ada tabel agregat baru. Perhitungan bersumber dari `inventory_items` dan `events`. Migration 015 yang direvisi menambahkan indeks pendukung agar pembacaan event dan tanggal asal lebih efisien.
