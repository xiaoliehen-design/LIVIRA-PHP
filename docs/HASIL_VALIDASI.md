# Hasil Validasi LIVIRA PHP

Tanggal validasi paket: 21 Juli 2026.

## Pemeriksaan yang lulus

- Seluruh berkas PHP pada `src`, `public`, `resources/views`, dan `tests` lolos `php -l`.
- JavaScript utama lolos `node --check`.
- Router parameter dinamis berfungsi.
- CAPTCHA menghasilkan challenge, memvalidasi jawaban, kedaluwarsa, dan hanya dapat digunakan sekali.
- DemoStore memuat seluruh tipe inventory.
- Normalisasi nomor kontainer berjalan.
- Action inventory memperbarui tahapan sesuai aksi.
- Form pencacahan dapat disimpan tanpa `inventory_ids[]` dari picker umum.
- Target pencacahan menggunakan ID inventory utama dan bukan `physical_unit_id`.
- Pencacahan FCL memperbarui seluruh uraian lama dan membuat satu baris inventory untuk setiap uraian baru.
- Uraian baru tidak menambah okupansi kontainer karena memakai `occupancy_primary=false`.
- Role tanpa pengguna dapat dihapus; role yang masih digunakan ditolak.
- Ekspor XLSX dapat dibaca kembali.
- Ekspor performa menghasilkan sheet `Ringkasan` dan `Rincian`.
- Kernel PHP melayani `/healthz` dan halaman login.
- Alias data template dan konversi acronym berjalan.
- Tidak terdapat file `.go`, `go.mod`, atau `go.sum`.
- Smoke test HTTP halaman utama, admin, proses, rekonsiliasi, laporan, ekspor, dan mutasi demo telah dijalankan tanpa fatal error PHP.

## Batas validasi

Validasi lokal tidak menggunakan credential Supabase produksi. Karena itu, sebelum mengganti layanan produksi, jalankan checklist pada `docs/VALIDASI_STAGING.md` dengan project Supabase staging atau salinan database. Dockerfile telah diperiksa secara statis, tetapi image Docker tidak dibangun di lingkungan validasi karena Docker CLI tidak tersedia.

## Validasi tambahan v1.0.4

- Total 23 pemeriksaan otomatis lulus.
- Alur pencacahan diuji melalui handler HTTP aplikasi dan DemoStore.
- Implementasi Store produksi diuji terhadap server REST lokal yang meniru GET/PATCH/POST Supabase.

## Validasi tambahan v1.0.2

- Apache front-controller test: `GET /healthz` = 200.
- Apache front-controller test: `GET /login` = 200.
- Apache front-controller test: `HEAD /` = 303 menuju login.
- Apache static file test: `HEAD /assets/app.css` = 200.
