# LIVIRA PHP

**Rilis saat ini: 1.0.6**

**LIVIRA — Layanan Inventori, Verifikasi, dan Integrasi** dalam implementasi PHP 8.2+ dengan Supabase sebagai Auth, PostgreSQL/API, RPC, dan Storage.

Paket ini adalah aplikasi **PHP-only**. Tidak ada source Go, `go.mod`, `go.sum`, binary Go, atau kebutuhan runtime Go.

## Fitur yang dipertahankan

- Login administrator dan pengguna Supabase, CAPTCHA sekali pakai, OTP pendaftaran, lupa password, serta idle logout 30 menit.
- Persetujuan/penolakan/hapus pengguna dan role dengan hak akses granular.
- Role hanya dapat dihapus ketika tidak digunakan pengguna.
- Dashboard seluruh kantor, TPS, seluruh TPP, per TPP, BTD/BDN/BMMN/barang titipan, kapasitas, YOR/SOR, dan performa.
- Inventory BTD, BDN, BMMN, barang titipan, FCL/LCL, multi-kontainer dan multi-rincian barang.
- Action pemindahan, pemberitahuan, pencacahan multi-uraian per kontainer, request/penelitian PFPD, penetapan/peruntukan BMMN, pengeluaran, dan bongkar/muat tanpa mengubah status barang.
- Proses lelang, pemusnahan, hibah/PSP, pengalihan hasil lelang, history, validasi transisi status, serta hasil lelang per ND penjadwalan.
- Rekonsiliasi fisik serta perubahan data barang dengan nilai sebelum/sesudah dan audit.
- Upload Excel massal BTD/BDN/barang titipan, template Excel yang konsisten dengan importer, upload dokumen private Supabase Storage, pencarian, pagination, notifikasi, dan ekspor CSV/XLS/XLSX.
- Parameter sistem dan TPP yang dapat dikelola administrator.

## Arsitektur

```text
public/index.php            Front controller PHP
src/App.php                 Routing dan controller aplikasi
src/Supabase/               Auth, REST/RPC, Storage, dan demo store
src/Security/               Session, CAPTCHA, rate limiter
src/Http/                   Request, response, router, middleware
resources/views/            Tampilan PHP hasil konversi template LIVIRA
public/assets/              CSS, JavaScript, favicon, template Excel
migrations/                 SQL database LIVIRA yang sudah ada
```

Tidak ada dependency runtime framework. Composer hanya digunakan untuk metadata/autoload opsional; aplikasi tetap dapat dijalankan langsung dengan PHP.

## Menjalankan secara lokal

Persyaratan minimum:

- PHP 8.2 atau lebih baru
- extension JSON dan Filter
- `mbstring` dan `zip` direkomendasikan
- command `zip`/`unzip` dapat menjadi fallback untuk XLSX

```bash
cp .env.example .env
```

Untuk preview lokal tanpa Supabase, ubah:

```env
APP_ENV=development
DEMO_MODE=true
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin-demo-only
SESSION_SECRET=local-secret-yang-panjang-minimal-32-karakter
```

Lalu jalankan:

```bash
php -S 127.0.0.1:8080 public/router.php
```

Buka `http://127.0.0.1:8080`.

## Menggunakan Supabase LIVIRA yang sekarang

Gunakan project Supabase yang sama. Tidak perlu memindahkan data, Auth user, Storage, RPC, atau menjalankan reset database.
Rilis 1.0.6 menggunakan kolom view `disposition_details.inventory_item_type` yang tersedia pada setup database LIVIRA dan memperbaiki template upload BTD/BDN; tidak memerlukan migration tambahan.

```env
APP_ENV=production
DEMO_MODE=false
PUBLIC_BASE_URL=https://domain-livira-anda
SESSION_SECRET=secret-random-minimal-32-karakter
SUPABASE_URL=https://PROJECT_REF.supabase.co
SUPABASE_ANON_KEY=...
SUPABASE_SERVICE_ROLE_KEY=...
SUPABASE_STORAGE_BUCKET=livira-documents
```

`SUPABASE_SERVICE_ROLE_KEY` hanya boleh berada di environment backend. Jangan masukkan nilai asli ke GitHub atau JavaScript.

## Deploy ke Render dari GitHub

1. Upload isi folder ini ke repository GitHub baru.
2. Di Render pilih **New → Blueprint** dan hubungkan repository; `render.yaml` akan membuat Web Service Docker **Free** di region Singapore.
3. Sebelum menekan deploy, pastikan estimasi biaya menunjukkan **$0/month**.
4. Isi environment yang bertanda `sync: false`.
5. Pastikan `PUBLIC_BASE_URL` memakai URL Render/custom domain final.
6. Deploy dan periksa `/healthz`.

Alternatifnya, buat **Web Service → Docker** secara manual dengan repository yang sama. Docker menjalankan Apache + PHP pada port `10000`.

Panduan rinci tersedia di [docs/DEPLOY_RENDER.md](docs/DEPLOY_RENDER.md).

## Database

Folder `migrations/` dipertahankan untuk dokumentasi/setup database baru. Untuk database Supabase produksi yang sudah berisi LIVIRA:

- jangan jalankan `01_SETUP_DATABASE_BARU_KOSONG...` kembali;
- jangan jalankan `02_RESET_SEMUA_DATA_BARANG...`;
- cukup arahkan environment PHP ke project Supabase yang sudah digunakan versi sebelumnya.

## Validasi

```bash
./scripts/validate.sh
```

Validasi mencakup:

- lint seluruh source PHP dan view;
- pemeriksaan JavaScript;
- router dinamis;
- CAPTCHA sekali pakai;
- operasi inventory, termasuk pencacahan FCL dengan uraian lama dan uraian baru;
- normalisasi nomor kontainer;
- penghapusan role kosong dan penolakan role terpakai;
- ekspor/baca ulang XLSX;
- kernel PHP, health check, halaman login, dan pemutusan sesi logout;
- verifikasi bahwa paket tidak mengandung source/runtime Go;
- pengujian regresi tombol logout dan idle logout;
- pengujian form pencacahan tanpa `inventory_ids[]`, target inventory utama, dan penyimpanan multi-uraian;
- pengujian tautan unduhan template BTD/BDN pada path utama dan kompatibilitas;
- pengujian parser XLSX, satu baris contoh, serta import BTD dan BDN secara end-to-end.

## Keamanan produksi

- `DEMO_MODE=true` ditolak pada `APP_ENV=production`.
- Konfigurasi Supabase wajib lengkap pada production.
- Session cookie ditandatangani, HttpOnly, SameSite, dan Secure pada HTTPS.
- CSRF diwajibkan pada seluruh mutasi.
- CAPTCHA sekali pakai dan login rate limiting aktif.
- Unduhan dokumen memeriksa cakupan inventory dan izin proses.
- Mutasi dan ekspor dicatat ke `audit_logs` tanpa menyimpan isi form sensitif.
- Content Security Policy dan header keamanan dikirim oleh aplikasi/Apache.

## Catatan pengujian produksi

Paket telah diuji secara lokal dengan PHP dan demo store. Koneksi live ke Supabase produksi tidak dapat diuji tanpa credential milik Anda. Lakukan deploy staging terlebih dahulu dan jalankan checklist pada [docs/VALIDASI_STAGING.md](docs/VALIDASI_STAGING.md) sebelum memindahkan domain produksi.
