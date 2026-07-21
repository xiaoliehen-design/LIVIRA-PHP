# Revisi Keamanan dan Performa — Migration 016

Dokumen ini berlaku untuk deployment yang **sudah menjalankan migration 015**. Jangan jalankan ulang migration 015. Backup database, jalankan migration berikut terlebih dahulu, lalu deploy source code terbaru:

```text
migrations/016_security_performance_hardening.sql
```

## Perubahan keamanan

1. **Konfigurasi production fail-closed**
   - Aplikasi menolak `DEMO_MODE=true`, URL non-HTTPS, secret sesi pendek/default, password admin lemah, dan konfigurasi bucket kosong pada production.
   - Akun admin lokal tidak memiliki kredensial production bawaan dan dapat dinonaktifkan dengan mengosongkan kedua variabel admin.
   - Bila akun lokal digunakan untuk bootstrap/darurat, rotasi username/password membatalkan seluruh sesi admin lokal lama setelah restart.

2. **Pencabutan sesi otomatis**
   - `app_users.session_version` diperiksa pada setiap request pengguna Supabase.
   - Perubahan role, permissions, status role, verifikasi email, atau persetujuan akun langsung membatalkan sesi dengan hak lama.

3. **Pembatasan autentikasi dan request**
   - Login, pendaftaran, verifikasi OTP, dan kirim ulang OTP memiliki rate limit berbasis IP serta identitas.
   - Ukuran body request dibatasi sebelum multipart diproses.
   - Header CSP, HSTS production, frame denial, no-sniff, COOP/CORP, dan `Cache-Control: no-store` diterapkan.

4. **Dokumen privat**
   - Lampiran baru disimpan pada bucket privat `livira-documents`; tabel hanya menyimpan metadata, path, ukuran, dan SHA-256.
   - Lampiran Base64 dari migration 015 tetap dapat dibaca.
   - Download memeriksa relasi event–inventory–permission sebelum file diambil.

5. **Audit keamanan**
   - Tabel `audit_logs` mencatat actor, tindakan, objek, outcome, IP, user-agent, request ID, metadata, dan waktu.
   - Audit mencakup autentikasi, pembuatan/penghapusan inventory, action inventory/proses, rekonsiliasi, impor, ekspor laporan, unduh dokumen, role, pengguna, parameter, serta kapasitas.

6. **Konsistensi workflow**
   - Penetapan multi-barang, mulai proses, action proses, update inventory, dan pencatatan timeline dilakukan dalam transaksi PostgreSQL yang sama.
   - Optimistic locking menolak penyimpanan jika data telah diubah pengguna lain sejak popup dibuka.

## Perubahan performa

- Daftar lelang/musnah/hibah memakai satu view join `disposition_details`, bukan satu request inventory untuk setiap proses.
- Performa tahunan tidak dihitung saat dashboard pertama kali dibuka; data dimuat ketika popup diminta. RPC `livira_performance_source` hanya mengirim barang dan event relevan pada periode terpilih, bukan seluruh inventory/timeline.
- Pencarian memakai `inventory_items.search_text` dan indeks GIN trigram.
- Notifikasi operasional dihitung melalui RPC agregasi, bukan mengunduh ribuan inventory pada setiap halaman.
- Parameter dinamis di-cache selama lima menit dan langsung diinvalidation setelah admin mengubahnya.
- Daftar inventory, pencarian detail, dan daftar lelang/musnah/hibah memakai count serta pagination di database.
- Dashboard proses tahunan dan ringkasan pelaporan dihitung melalui RPC agregasi; halaman tidak lagi mengunduh seluruh riwayat hanya untuk membuat grafik/kartu statistik.
- Halaman pelaporan inventory mengambil satu halaman data, sementara total jumlah, nilai, posisi TPP, dan status dihitung di database.
- Indeks ditambahkan untuk pagination, timeline, proses, dokumen, pencarian, dan agregasi.

## Langkah deployment

1. Backup database Supabase.
2. Siapkan environment deployment berikut, tetapi jangan alihkan traffic ke source baru sebelum migration selesai:

```text
APP_ENV=production
DEMO_MODE=false
PUBLIC_BASE_URL=https://domain-anda
SESSION_SECRET=<acak-minimal-32-karakter>
# Opsional break-glass; isi keduanya atau kosongkan keduanya.
ADMIN_USERNAME=<username-admin-lokal>
ADMIN_PASSWORD=<minimal-16-karakter-kuat>
SUPABASE_URL=...
SUPABASE_ANON_KEY=...
SUPABASE_SERVICE_ROLE_KEY=...
SUPABASE_STORAGE_BUCKET=livira-documents
```

3. Jalankan migration 016 melalui SQL Editor dan pastikan transaksi berhasil.
4. Deploy/restart source code terbaru. Migration 016 tetap kompatibel dengan source lama selama masa rollout singkat.
5. Uji login, perubahan role, pencarian, satu action inventory, satu action proses, upload/unduh dokumen, dan ekspor laporan.
6. Jalankan Supabase Security Advisor dan Performance Advisor.

## Batasan yang tetap perlu dikendalikan

- Rate limiter bawaan tersimpan per-instance. Deployment multi-instance sebaiknya menambahkan rate limiting terpusat melalui reverse proxy/WAF atau Redis.
- File sudah dibatasi tipe/ukuran dan diverifikasi checksum, tetapi pemeriksaan malware memerlukan layanan scanner eksternal.
- MFA admin sebaiknya diaktifkan melalui penyedia identitas/Supabase sebelum penggunaan lintas unit.
- Backup harus diuji melalui restore drill berkala; keberadaan backup saja belum menjamin pemulihan.
- Export sangat besar dan bulk action lintas ratusan item masih perlu diuji beban sesuai volume produksi nyata.
