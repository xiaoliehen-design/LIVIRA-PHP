# Revisi Laporan BTD Lengkap 022

Preset **Laporan BTD** kini tetap menggunakan satu baris per dokumen BTD yang telah dideduplikasi, tetapi menampilkan informasi yang lebih lengkap:

- nomor dan tanggal BTD;
- nomor BL;
- nomor, tanggal, dan pos manifest;
- jenis muatan;
- TPS asal dan TPP;
- status lokasi;
- nomor serta ukuran kontainer, atau volume LCL;
- uraian, jenis, kondisi, jumlah, dan satuan barang;
- jumlah rincian barang dan total nilai;
- pemilik/shipper/consignee;
- status barang dan status inventory.

Ekspor CSV dan Excel menggunakan 17 kolom yang sama dengan tabel. Nilai yang sama dalam satu dokumen dideduplikasi. Apabila satu dokumen memiliki lebih dari satu nilai yang sah, seluruh nilai unik ditampilkan dengan pemisah titik koma.

Revisi ini tidak memerlukan migration database baru karena seluruh kolom sumber sudah tersedia sejak migration sebelumnya.


> Catatan revisi 023: Tanggal BL ditambahkan setelah Nomor BL, sehingga tabel dan ekspor Laporan BTD sekarang terdiri atas 18 kolom.
