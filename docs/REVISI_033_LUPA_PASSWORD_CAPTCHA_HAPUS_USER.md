# Revisi 033 — Lupa Password, CAPTCHA Login, dan Hapus User

## Ringkasan perubahan

### 1. Lupa password dengan OTP email

- Tautan **Lupa password?** tersedia pada halaman login.
- Pengguna memasukkan email yang sama dengan email pendaftaran.
- Supabase mengirim OTP pemulihan 6 digit.
- Pengguna mengisi email, OTP, password baru, dan konfirmasi password.
- Backend memverifikasi OTP bertipe `recovery`, menggunakan access token pemulihan hanya untuk menyimpan password baru, lalu tidak menyimpan atau mengekspos token tersebut ke browser.
- Pesan permintaan OTP tidak mengungkap apakah suatu email terdaftar.
- Permintaan dan verifikasi OTP memiliki rate limit terpisah serta jejak audit.

### 2. CAPTCHA pada login

- Login wajib melewati CAPTCHA visual 5 karakter.
- Jawaban CAPTCHA dienkripsi dan diautentikasi menggunakan kunci turunan `SESSION_SECRET`.
- Token CAPTCHA kedaluwarsa setelah 5 menit dan tidak disimpan di database.
- Tombol **Kode baru** mengganti tantangan tanpa memuat ulang halaman.
- Kegagalan CAPTCHA tetap dihitung sebagai percobaan login sehingga perlindungan CAPTCHA dan rate limit berjalan bersama.
- Implementasi tidak memerlukan akun, site key, atau secret key CAPTCHA pihak ketiga.

### 3. Hapus user oleh admin

- Tombol **Hapus user** tersedia pada setiap baris menu **Setujui Pendaftaran**, baik untuk akun pending, ditolak, maupun disetujui.
- Browser meminta konfirmasi sebelum penghapusan permanen.
- Backend menggunakan `SUPABASE_SERVICE_ROLE_KEY` untuk menghapus identitas dari Supabase Auth.
- Relasi `app_users.auth_user_id` yang menggunakan `ON DELETE CASCADE` otomatis menghapus profil pendaftaran terkait.
- Sesi aplikasi user yang dihapus langsung ditolak pada request berikutnya karena profil akses tidak lagi tersedia.
- Hasil penghapusan dicatat pada audit aplikasi.

## SQL database

Revisi 033 tidak memerlukan migration SQL baru. File setup database kosong sampai revisi 032 sudah menetapkan relasi berikut:

```sql
auth_user_id uuid not null unique references auth.users(id) on delete cascade
```

Untuk database lama, pastikan migration akses pengguna (`007_access_approval_parameters.sql`) pernah diterapkan sebelum memakai fitur penghapusan user.

## Konfigurasi wajib Supabase

Selain template **Confirm signup**, template **Reset Password** juga harus menampilkan `{{ .Token }}`.

1. Buka **Supabase → Authentication → Email Templates → Reset Password**.
2. Pastikan isi template memuat kode berikut:

   ```html
   <p style="font-size:32px;font-weight:700;letter-spacing:8px">{{ .Token }}</p>
   ```

3. Simpan template.
4. Pastikan custom SMTP aktif untuk penggunaan operasional.
5. Tidak ada environment variable baru. Pertahankan `SESSION_SECRET` acak minimal 32 karakter dan `SUPABASE_SERVICE_ROLE_KEY` hanya pada backend.

## Langkah deployment

1. Ganti source lama di GitHub dengan seluruh isi source revisi 033.
2. Jangan unggah file `.env`.
3. Tidak perlu menjalankan ulang file setup database atau migration lama.
4. Konfigurasikan template Reset Password seperti petunjuk di atas.
5. Tunggu Render selesai melakukan deployment.

## Pemeriksaan setelah deployment

1. Buka login dan pastikan CAPTCHA tampil.
2. Coba kode CAPTCHA salah; login harus ditolak.
3. Tekan **Kode baru**; gambar harus berubah tanpa reload halaman.
4. Pilih **Lupa password?**, masukkan email pendaftaran, lalu pastikan OTP 6 digit diterima.
5. Masukkan OTP dan password baru; login dengan password lama harus gagal, sedangkan password baru harus berhasil.
6. Buka **Setujui Pendaftaran**, hapus satu akun uji, lalu pastikan akun hilang dari daftar dan tidak dapat login.
7. Jangan menguji penghapusan menggunakan akun operasional yang masih diperlukan karena tindakan ini permanen.
