# Revisi Parameter Sistem Lengkap

Menu **Administrasi → Parameter Sistem** sekarang mengelola master dropdown berikut:

- Kategori BDN
- Jenis barang
- Satuan barang
- Jenis peruntukan BMMN
- TPS asal
- Nama TPP
- Jenis muatan
- Jenis pengeluaran
- Jenis serah terima Hibah/PSP

## Perilaku penghapusan

Tombol **Hapus dari dropdown** menonaktifkan parameter. Nilai yang sudah tersimpan pada data lama tidak dihapus. Parameter dapat diaktifkan kembali dari halaman yang sama.

Nama TPP dikelola melalui tabel `facilities`, bukan disalin ke `app_parameters`. TPP baru langsung tersedia pada formulir, filter, dan dashboard. Kapasitas yard dan shed TPP baru dimulai dari 0. TPP yang masih digunakan inventory aktif tidak dapat dinonaktifkan sampai barang dipindahkan atau diselesaikan.

## Dropdown yang tetap dikunci

Jenis inventory BTD/BDN/BMMN, status workflow, hasil lelang, dan status lartas tidak dijadikan parameter bebas karena nilainya dipakai oleh logika transisi proses.

## Database

Jalankan `migrations/009_expand_system_parameters.sql` setelah migration 008.
