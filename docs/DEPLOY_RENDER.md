# Deploy LIVIRA PHP ke Render

## 1. Upload ke GitHub

Upload seluruh isi ZIP ke root repository. Jangan upload file `.env`.

File penting deployment:

- `Dockerfile`
- `render.yaml`
- `public/index.php`
- `src/`
- `resources/`
- `public/assets/`
- `migrations/`

## 2. Buat service

Pilihan paling mudah:

1. Render Dashboard → **New** → **Blueprint**.
2. Pilih repository LIVIRA PHP.
3. Render membaca `render.yaml` dan membuat service `livira-php` pada instance **Free** di region Singapore.
4. Pada halaman estimasi biaya, pastikan total tertulis **$0/month**. Jika tertulis $7/month, batalkan deployment dan pastikan `render.yaml` memuat `plan: free`.

Pilihan manual:

1. Render Dashboard → **New** → **Web Service**.
2. Hubungkan repository.
3. Runtime: **Docker**.
4. Health Check Path: `/healthz`.
5. Dockerfile Path: `./Dockerfile`.
6. Pada **Instance Type**, pilih **Free**.

## 3. Environment production

Isi nilai berikut di **Environment**:

```env
APP_NAME=LIVIRA
APP_ENV=production
DEMO_MODE=false
IDLE_TIMEOUT_SECONDS=1800
PUBLIC_BASE_URL=https://URL-SERVICE-ANDA.onrender.com
SESSION_SECRET=RANDOM_SECRET_MINIMAL_32_KARAKTER
ADMIN_USERNAME=admin
ADMIN_PASSWORD=PASSWORD_ADMIN_MINIMAL_16_KARAKTER
SUPABASE_URL=https://PROJECT_REF.supabase.co
SUPABASE_ANON_KEY=...
SUPABASE_SERVICE_ROLE_KEY=...
SUPABASE_STORAGE_BUCKET=livira-documents
```

Administrator lokal bersifat opsional. `ADMIN_USERNAME` dan `ADMIN_PASSWORD` dapat sama-sama dikosongkan bila seluruh admin dikelola melalui Supabase.

## 4. Setelah deploy

Periksa:

- `https://URL-ANDA/healthz` mengembalikan `status: ok` dan `app: LIVIRA PHP`.
- halaman login tampil;
- CAPTCHA dapat diperbarui;
- login admin dan login user Supabase berhasil;
- dashboard membaca database yang sama;
- upload/download dokumen mengakses bucket yang benar.

## 5. Cutover aman

Gunakan service baru, misalnya `livira-php-staging`, tanpa menghapus service lama. Setelah checklist staging selesai:

1. hentikan mutasi pada service lama;
2. lakukan smoke test final pada service PHP;
3. arahkan custom domain ke service PHP;
4. pertahankan service lama sementara sebagai rollback;
5. jangan menjalankan reset SQL saat cutover.

## 6. Troubleshooting route 404 setelah service Live

Jika log menunjukkan root `/` mengembalikan redirect `303`, tetapi `/login` mengembalikan `404`, berarti Apache belum meneruskan route aplikasi ke front controller PHP.

Paket v1.0.2 memperbaikinya melalui VirtualHost pada `Dockerfile` dengan:

- `DocumentRoot /var/www/html/public`;
- `AllowOverride All`;
- `FallbackResource /index.php`;
- dukungan request `HEAD` sebagai route `GET`.

Setelah mengunggah paket v1.0.2 ke GitHub, buka service Render lalu pilih **Manual Deploy → Clear build cache & deploy** agar image lama tidak digunakan kembali.
