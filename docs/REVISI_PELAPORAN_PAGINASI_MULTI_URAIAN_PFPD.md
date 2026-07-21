# Revisi pelaporan, tabel, multi-uraian, dan penelitian PFPD

1. Jalankan `migrations/012_reporting_pagination_multi_goods_pfpd.sql` setelah migration 011.
2. Deploy ulang aplikasi Golang.
3. Pelaporan tidak lagi menampilkan pilihan urutan manual dan menyediakan tombol ekspor CSV dan Excel pada area filter.
4. Tabel Inventory, Lelang, Musnah, Hibah, dan Pelaporan memiliki scrollbar horizontal di bagian atas, pilihan 10/20/50/100 baris, serta tombol halaman sebelumnya/berikutnya.
5. Pencacahan dapat membentuk beberapa baris uraian untuk satu unit fisik. Nomor kontainer tetap sama dan kapasitas YOR/SOR hanya dihitung satu kali.
6. Penelitian PFPD dipilih berdasarkan nomor request. Daftar request memiliki pencarian nomor request; setelah request dibuka, HS code, nilai, dan status lartas diisi untuk setiap uraian barang.
7. Pertanyaan apakah penelitian PFPD diperlukan dihapus dari action Pencacahan. Hasil pencacahan tetap dapat diteruskan melalui action Request Penelitian PFPD.
