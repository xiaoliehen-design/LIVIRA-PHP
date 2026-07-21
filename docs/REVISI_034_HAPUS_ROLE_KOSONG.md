# Revisi 034 — Hapus Role Tanpa Pengguna

## Perubahan

- Setiap kartu role menampilkan jumlah akun yang masih mereferensikan role tersebut.
- Tombol **Hapus role** hanya tampil jika jumlah pengguna adalah 0.
- Admin memperoleh dialog konfirmasi sebelum penghapusan permanen dikirim.
- Endpoint penghapusan menggunakan `POST`, validasi sesi, permission `admin.roles`, dan token CSRF.
- Penghapusan sukses maupun gagal dicatat ke audit log dengan action `role.delete`.

## Perlindungan data

Backend tidak mempercayai jumlah pengguna yang ditampilkan pada halaman. Saat penghapusan diproses, database menjalankan `DELETE` langsung pada `app_roles`. Relasi `app_users.role_id` sudah menggunakan `ON DELETE RESTRICT`, sehingga role yang masih digunakan tetap ditolak secara atomik walaupun:

- pengguna baru memperoleh role setelah halaman admin dibuka;
- tombol/endpoint dipanggil secara langsung;
- dua admin melakukan perubahan pada waktu yang hampir bersamaan.

Jika role masih digunakan, admin menerima pesan untuk memindahkan role pengguna terlebih dahulu. Role tidak berubah dan tidak ada akun yang kehilangan referensi role.

## Deployment

Revisi ini tidak memerlukan migration SQL baru. Unggah seluruh source terbaru ke GitHub dan tunggu Render menyelesaikan deployment. Struktur database pada setup gabungan sudah memiliki foreign key yang diperlukan.

## Pemeriksaan cepat

1. Buka **Admin → Role & Hak Akses**.
2. Pastikan kartu role menampilkan badge jumlah pengguna.
3. Buat role uji tanpa pengguna dan pastikan tombol **Hapus role** tampil.
4. Hapus role uji, setujui konfirmasi, lalu pastikan kartu role hilang.
5. Periksa role yang digunakan akun; tombol hapus tidak boleh tampil dan keterangannya harus menyebut jumlah pengguna.
