# Panduan Upgrade LIVIRA

Panduan ini digunakan untuk mengganti versi LIVIRA yang sudah berada di GitHub dan Render dengan source terbaru.

## Mode demo

1. Ekstrak ZIP versi lengkap.
2. Unggah seluruh isi folder project ke repository GitHub, termasuk `.github`, `cmd`, `internal`, `migrations`, dan file pada root.
3. Pilih **replace/overwrite** untuk file lama, kemudian commit ke branch `main`.
4. Render akan melakukan deployment ulang otomatis apabila Auto-Deploy aktif.
5. Apabila terdapat environment variable `APP_NAME`, gunakan nilai `LIVIRA`.

Mode demo tidak memerlukan migration SQL. Data demo tersimpan di memori dan kembali ke data contoh ketika service dimulai ulang.

## Database Supabase yang sudah sampai migration 016

1. Buat backup database terlebih dahulu.
2. Buka **Supabase > SQL Editor**.
3. Salin dan jalankan seluruh isi file:

   `migrations/017_transfer_lelang_rekonsiliasi_perubahan_data.sql`

4. Pastikan query selesai tanpa error.
5. Unggah source terbaru ke GitHub dan commit ke branch `main`.
6. Tunggu deployment Render selesai dan berstatus **Live**.

Migration 017 diperlukan untuk:

- memindahkan hasil lelang berstatus **Tidak Laku** ke proses Pemusnahan atau Hibah/PSP;
- menutup proses lelang lama secara atomik agar tidak lagi muncul di halaman utama Lelang;
- menambahkan rekonsiliasi **Perubahan data barang**;
- memperbarui data barang, timeline, dan data proses secara atomik sambil menyimpan jejak audit.

Jangan menjalankan file setup database kosong pada database operasional yang sudah berisi data.

## Database Supabase baru dan masih kosong

Untuk database yang benar-benar baru dan belum memiliki tabel maupun data LIVIRA:

1. Buka **Supabase > SQL Editor**.
2. Salin dan jalankan seluruh isi file:

   `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`

3. Setelah berhasil, lanjutkan pengaturan environment variable Supabase pada Render.
4. Konfigurasikan template **Confirm signup** dan **Reset Password** Supabase agar keduanya memuat `{{ .Token }}` sebagai OTP 6 digit. Petunjuknya tersedia di `docs/KONFIGURASI_OTP_DAN_AKSES.md`.

File setup gabungan tersebut sudah mencakup struktur database sampai migration 017. Jangan menjalankan migration lama satu per satu setelah file gabungan berhasil dijalankan.

## Pemeriksaan setelah deployment

Lakukan pemeriksaan berikut:

- Dashboard langsung menampilkan performa tahun berjalan saat pertama dibuka.
- Halaman utama Lelang tidak menampilkan barang yang sudah Laku atau sudah dialihkan ke Pemusnahan/Hibah.
- History Lelang menampilkan proses yang Laku dan proses Tidak Laku yang sudah dialihkan.
- Barang lelang berstatus Tidak Laku dapat dicari pada Action Pemusnahan dan Hibah/PSP.
- Setelah proses pengalihan disimpan, barang menghilang dari halaman utama Lelang dan muncul pada proses tujuan.
- Halaman utama Pemusnahan tidak menampilkan proses yang sudah selesai, sedangkan History Pemusnahan menampilkannya.
- Popup **Catat rekonsiliasi** memiliki tiga opsi: barang tercatat tetapi tidak ditemukan, barang ditemukan tetapi belum tercatat, dan perubahan data barang.
- Perubahan data barang dapat mengoreksi identitas, uraian, lokasi, kontainer, BCF, penetapan, dokumen timeline, serta data proses terkait.
- Opsi alasan perubahan hanya memuat **Kesalahan input** dan **Error pada saat pengisian awal**, serta wajib dipilih sebelum penyimpanan.
- Setelah koreksi disimpan, timeline barang memuat jejak **Perubahan data barang** beserta alasan dan pengguna yang melakukan perubahan.
- Halaman login, pendaftaran, dan OTP tampil normal pada desktop maupun perangkat seluler.
- Halaman login menampilkan CAPTCHA 5 karakter, tombol **Kode baru** berfungsi, dan kode yang salah ditolak.
- Menu **Lupa password** mengirim OTP ke email pendaftaran dan dapat menyimpan password baru setelah OTP valid.
- Menu **Setujui Pendaftaran** menampilkan tombol **Hapus user**; user yang dihapus hilang dari daftar dan tidak dapat login kembali.
- Menu **Role & Hak Akses** menampilkan jumlah pengguna pada setiap role. Role dengan 0 pengguna memiliki tombol **Hapus role**, sedangkan role yang masih digunakan tidak dapat dihapus.

Jalankan pengujian utama setelah deployment dengan satu data uji sebelum digunakan pada data operasional dalam jumlah besar.
