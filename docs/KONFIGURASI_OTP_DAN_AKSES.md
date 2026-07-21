# Konfigurasi OTP, Reset Password, Persetujuan Pendaftaran, Role, dan Parameter

Panduan ini berlaku untuk source LIVIRA yang sudah memiliki migration `007_access_approval_parameters.sql`.

## 1. Jalankan migration database

Buka **Supabase → SQL Editor**, lalu jalankan:

```text
migrations/007_access_approval_parameters.sql
```

Jalankan setelah migration 006. Migration ini membuat:

- `app_users` untuk status OTP, persetujuan, dan role pengguna;
- `app_roles` untuk nama role dan kombinasi hak akses;
- `app_parameters` untuk kategori BDN, jenis barang, satuan, peruntukan BMMN, TPS asal, jenis muatan, jenis pengeluaran, dan jenis serah terima; nama TPP tetap memakai tabel `facilities`;
- trigger pada `auth.users` agar pendaftaran baru otomatis masuk ke antrean admin.

Akun Supabase Auth yang sudah ada sebelum migration akan dimasukkan sebagai `pending`. Admin perlu meninjau dan menyetujuinya sebelum akun tersebut dapat masuk melalui source baru.

## 2. Aktifkan konfirmasi email

Pada dashboard Supabase:

1. Buka **Authentication → Providers → Email**.
2. Pastikan **Confirm email** dalam keadaan aktif.
3. Buka **Authentication → URL Configuration**.
4. Isi **Site URL** dengan alamat aplikasi yang sama dengan `PUBLIC_BASE_URL`.

## 3. Ubah email konfirmasi menjadi OTP 6 digit

Buka **Authentication → Email Templates → Confirm signup**. Pastikan isi email menggunakan variabel berikut:

```text
{{ .Token }}
```

Contoh template:

```html
<h2>Konfirmasi pendaftaran LIVIRA</h2>
<p>Masukkan kode OTP berikut pada halaman konfirmasi:</p>
<p style="font-size:32px;font-weight:700;letter-spacing:8px">{{ .Token }}</p>
<p>Jangan berikan kode ini kepada orang lain.</p>
```

Apabila template masih hanya menggunakan `{{ .ConfirmationURL }}`, pengguna akan menerima tautan, bukan kode yang diminta oleh halaman OTP aplikasi.

Untuk deployment operasional, gunakan custom SMTP pada pengaturan Authentication agar email OTP tidak bergantung pada layanan email bawaan untuk kebutuhan pengujian.

## 4. Konfigurasikan OTP reset password

Buka **Authentication → Email Templates → Reset Password**. Ganti template agar menampilkan variabel OTP berikut:

```text
{{ .Token }}
```

Contoh template:

```html
<h2>Reset password LIVIRA</h2>
<p>Masukkan kode OTP berikut pada halaman reset password:</p>
<p style="font-size:32px;font-weight:700;letter-spacing:8px">{{ .Token }}</p>
<p>Kode bersifat rahasia dan memiliki masa berlaku terbatas.</p>
<p>Abaikan email ini jika Anda tidak meminta perubahan password.</p>
```

Template yang hanya memuat `{{ .ConfirmationURL }}` akan mengirim tautan, bukan OTP 6 digit yang diminta halaman LIVIRA.

Alur pemulihan password:

1. Pengguna memilih **Lupa password?** pada halaman login.
2. Pengguna memasukkan email yang dipakai ketika mendaftar.
3. LIVIRA meminta Supabase mengirim OTP pemulihan tanpa mengungkap apakah email terdaftar.
4. Pengguna memasukkan email, OTP 6 digit, password baru, dan konfirmasi password.
5. Backend memverifikasi OTP bertipe `recovery`, memakai access token pemulihan hanya untuk memperbarui password, lalu tidak menyimpan atau mengirim token tersebut ke browser.
6. Pengguna kembali ke halaman login dan masuk memakai password baru.

Reset password ini hanya berlaku bagi akun email Supabase. Password administrator lokal/break-glass diubah melalui environment `ADMIN_PASSWORD` pada Render atau server deployment.

## 5. Alur pendaftaran pengguna

1. Pengguna membuka **Daftar dengan email**.
2. Pengguna mengisi nama, email, dan password.
3. Supabase mengirim OTP 6 digit ke email.
4. Pengguna memasukkan OTP pada halaman **Konfirmasi OTP email**.
5. Status email berubah menjadi terverifikasi, tetapi pengguna belum dapat masuk.
6. Admin membuka **Administrasi → Setujui Pendaftaran**.
7. Admin memilih role lalu menekan **Setujui**.
8. Pengguna dapat login menggunakan email dan password sesuai hak akses role.

Pendaftaran yang belum mengonfirmasi OTP tidak dapat disetujui. Pendaftaran yang masih `pending` atau telah `rejected` juga tidak dapat masuk.

Admin juga dapat menekan **Hapus user** untuk menghapus pendaftar atau pengguna yang sudah disetujui. Tindakan ini menghapus identitas Supabase Auth dan profil `app_users` secara permanen; pengguna harus mendaftar ulang jika membutuhkan akses kembali.

## 6. Membuat role custom

Buka **Administrasi → Role & Hak Akses**. Admin dapat menentukan nama role, deskripsi, dan kombinasi akses.

Contoh:

- **Petugas Lelang BMMN**: Lihat/Kelola Lelang, Akses BMMN, dan Pencarian Detail.
- **Viewer BDN**: Lihat Dashboard, Lihat Inventory, Akses BDN, dan Pelaporan.
- **Petugas Hibah/PSP**: Lihat/Kelola Hibah/PSP, Akses BMMN, dan Pencarian Detail.
- **Petugas Pemusnahan**: Lihat/Kelola Pemusnahan dan cakupan jenis barang yang diperlukan.

Hak **Kelola** otomatis memasukkan hak **Lihat** untuk menu yang sama. Pengguna membaca hak akses terbaru saat login; setelah role diubah, minta pengguna logout dan login kembali.

Role yang masih digunakan oleh akun aktif tidak dapat dinonaktifkan.

## 7. Mengelola parameter dropdown

Buka **Administrasi → Parameter Sistem**. Kelompok parameter yang dapat dikelola:

- Kategori BDN;
- Jenis barang;
- Satuan barang;
- Jenis peruntukan BMMN;
- TPS asal;
- Nama TPP;
- Jenis muatan;
- Jenis pengeluaran;
- Jenis serah terima Hibah/PSP.

Untuk jenis pengeluaran, pilih cakupan BTD, BDN, dan/atau BMMN. Parameter baru langsung dipakai oleh dropdown dan validasi backend.

Tombol **Hapus dari dropdown** melakukan penonaktifan, bukan penghapusan fisik. Nilai lama tetap tersimpan pada inventory dan timeline yang sudah menggunakan parameter tersebut. Parameter dapat diaktifkan kembali kapan saja.

## 8. CAPTCHA login

CAPTCHA login dibuat dan diverifikasi oleh backend LIVIRA. Jawabannya dienkripsi menggunakan kunci turunan `SESSION_SECRET`, kedaluwarsa setelah 5 menit, tidak disimpan di database, dan dapat diperbarui melalui tombol **Kode baru**. Tidak ada environment variable CAPTCHA pihak ketiga yang perlu ditambahkan.

Pastikan `SESSION_SECRET` pada production berupa nilai acak minimal 32 karakter. CAPTCHA melengkapi, bukan menggantikan, pembatasan percobaan login yang sudah diterapkan aplikasi.

## 9. Pemeriksaan setelah deployment

- Daftar akun mengirim OTP 6 digit ke email.
- OTP benar mengarahkan pengguna ke status menunggu admin.
- Login sebelum persetujuan ditolak dengan pesan yang sesuai.
- Halaman login menampilkan CAPTCHA dan menolak kode salah atau kedaluwarsa.
- Tombol **Kode baru** mengganti gambar dan token CAPTCHA tanpa memuat ulang halaman.
- Menu Lupa password mengirim OTP dari template **Reset Password**, dan OTP yang benar dapat menyimpan password baru.
- Menu admin hanya terlihat pada akun administrator utama.
- Admin dapat menyetujui pengguna sambil memilih role.
- Admin dapat menghapus user, dan user yang dihapus tidak lagi muncul di daftar serta tidak dapat login.
- Pengguna hanya melihat menu dan data sesuai hak akses serta cakupan BTD/BDN/BMMN.
- Parameter nonaktif tidak muncul lagi pada dropdown baru.
- `.github/workflows/ci.yml` terlihat di repository dan GitHub Actions lulus.
