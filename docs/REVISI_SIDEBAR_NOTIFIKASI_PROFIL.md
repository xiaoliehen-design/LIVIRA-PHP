# Revisi Sidebar, Notifikasi, dan Profil

Revisi ini tidak membutuhkan migration database baru. Source tetap menggunakan migration 001–008.

## Sidebar admin

- Bagian merek dan status database tetap terlihat.
- Daftar menu di tengah memiliki scroll vertikal sendiri.
- Scrollbar dibuat tipis agar tidak mengganggu tampilan.
- Pada layar kecil, sidebar tetap dapat dibuka dan ditutup melalui tombol menu.

## Pusat notifikasi

Tombol lonceng pada kanan atas sekarang membuka pusat notifikasi. Notifikasi dibuat dari data aktual yang dapat diakses oleh role pengguna, meliputi:

- pendaftaran yang sudah mengonfirmasi OTP dan siap disetujui admin;
- pendaftaran yang masih menunggu verifikasi email;
- BTD/BDN yang telah melewati 60 hari tanpa tindak lanjut;
- barang yang selesai lelang, musnah, hibah, atau PSP dan menunggu pengeluaran;
- BMMN yang masih menunggu peruntukan.

Setiap notifikasi dapat diklik dan mengarah langsung ke halaman tindakan atau laporan terkait. Jika tidak ada perhatian baru, panel menampilkan status terkendali.

## Menu profil kanan atas

Klik nama atau avatar pengguna untuk membuka menu akun. Menu berisi:

- **Buka profil** untuk menampilkan detail nama, email, role, status akun, keamanan sesi, dan daftar hak akses;
- **Logout** untuk keluar dari akun secara langsung.

Popup profil juga menyediakan tombol logout dan informasi bahwa sesi berakhir otomatis setelah 30 menit tanpa aktivitas.
