# LIVIRA — Layanan Inventori, Verifikasi, Integrasi, Rekonsiliasi, dan Analitik

**LIVIRA (Layanan Inventori, Verifikasi, Integrasi, Rekonsiliasi, dan Analitik)** adalah aplikasi pengelolaan **BTD, BDN, dan BMMN** dengan alur penyelesaian barang melalui **lelang, pemusnahan, atau hibah/PSP**. Backend, routing, autentikasi, validasi, dan integrasi database ditulis dalam bahasa Go. Tampilan menggunakan HTML template, CSS, dan JavaScript ringan yang disajikan langsung oleh server Go.

## Fitur utama

- Dashboard KPI dengan cakupan **seluruh Kantor Tanjung Priok**, masih di TPS, seluruh TPP, atau TPP tertentu. Detail per TPP dan YOR/SOR tetap menampilkan kondisi riil setiap TPP serta tidak berubah ketika cakupan KPI diganti.
- Inventory aktif dengan tab BTD/BDN/BMMN, pencarian, filter TPP/status, pengurutan penetapan terbaru dan nilai, serta halaman History untuk barang yang telah keluar.
- **Pencatatan BTD** dan Penetapan BDN mencatat apakah barang masih di TPS asal atau sudah berada di TPP. Pencatatan BTD mewajibkan nomor BL dan tanggal BL.
- Nomor penetapan menjadi referensi yang dapat diklik untuk membuka detail barang.
- Tombol **Action** sejajar dengan Pencatatan BTD/Penetapan BDN, dengan pencarian kontainer dan pemilihan banyak barang untuk satu dokumen/tahapan.
- Master dropdown operasional dikelola dinamis oleh admin: kategori BDN, jenis barang, satuan, jenis peruntukan BMMN, TPS asal, nama TPP, jenis muatan, jenis pengeluaran, dan jenis serah terima.
- Action untuk satu atau banyak barang mencakup pemindahan, bongkar/muat kontainer, pemberitahuan, pencacahan, request/penelitian PFPD, Penetapan BMMN, dua tahap peruntukan BMMN, dan pengeluaran sesuai jenis barang. Hak akses setiap action dapat diberikan secara terpisah.
- Perubahan BTD/BDN menjadi BMMN melalui action **Penetapan BMMN**; BMMN tidak dapat dibuat langsung.
- Nilai barang tampil di Inventory, Lelang, Pelaporan, dan detail barang.
- Menu **Pencarian Detail Barang** mencari inventory aktif maupun selesai berdasarkan dokumen, kontainer, manifest, status, TPP, jenis barang, peruntukan BMMN, dan rentang tanggal; hasil dapat dibuka untuk detail lengkap dan timeline pengerjaan.
- Status paling kanan dapat diklik untuk membuka timeline lengkap dengan timestamp, dokumen, catatan, petugas, serta tautan unduh dokumen PDF/gambar yang dilampirkan.
- Lelang memuat KEP Lelang, KEP HTL, jadwal, hasil laku/tidak laku, lelang penyesuaian, dan alokasi hasil; dashboard menampilkan tren nilai barang, HTL, nilai jual, jumlah tahun berjalan, dan proses aktif.
- Pemusnahan memuat KEP Musnah, BA Musnah, biaya, dan grafik jumlah/biaya; Hibah/PSP memuat BA Serah Terima serta grafik jumlah Hibah dibanding PSP.
- Menu Lelang, Pemusnahan, dan Hibah/PSP selalu memulai action pertama dengan mencari barang dari inventory aktif.
- Satu barang tidak dapat berada dalam dua proses penyelesaian aktif sekaligus.
- Barang berstatus Laku, BA Musnah, atau BA Serah Terima dapat dikeluarkan melalui action **Pengeluaran Barang** di Inventory. Setelah dokumen pengeluaran disimpan, barang menjadi tidak aktif, hilang dari menu operasional, dan berpindah ke History.
- Pelaporan kustom berdasarkan rentang tanggal, status aktif/selesai, TPP, lokasi, umur, nilai, jenis, status barang, dan peruntukan BMMN; seluruh hasil filter dan preset laporan dapat diekspor ke CSV UTF-8 maupun Excel `.xlsx`, sedangkan laporan performa kinerja tetap memakai Excel dengan sheet ringkasan dan rincian.
- Preset laporan mencakup **Laporan BTD lengkap per dokumen** (BL, manifest, TPS/TPP, kontainer/LCL, rincian barang, nilai, dan status), BTD/BDN berumur sekurangnya 60 hari yang belum ditindaklanjuti, potensi siap lelang berdasarkan nilai tertinggi, barang aktif per TPP, barang masih di TPS, BMMN menunggu peruntukan, riwayat selesai, rekonsiliasi, perubahan data barang, dan performa kinerja.
- Pendaftaran akun menggunakan email dan OTP 6 digit Supabase sebelum masuk ke antrean persetujuan admin.
- Halaman login dilindungi CAPTCHA visual 5 karakter yang terenkripsi dengan `SESSION_SECRET`, memiliki masa berlaku singkat, dapat diperbarui tanpa memuat ulang halaman, dan tetap dipadukan dengan pembatasan percobaan login.
- Menu **Lupa password** mengirim OTP 6 digit ke email yang digunakan saat pendaftaran. OTP diverifikasi oleh Supabase sebelum password baru disimpan, tanpa mengekspos access token pemulihan ke browser.
- Akun baru tidak dapat login sampai admin menyetujui pendaftaran dan menetapkan role aktif. Role pengguna yang telah disetujui tetap dapat diubah dari menu admin.
- Dari menu **Setujui Pendaftaran**, admin dapat menghapus user berstatus pending, ditolak, maupun disetujui. Penghapusan juga menghapus identitas Supabase Auth sehingga akun tidak dapat login kembali.
- Menu admin khusus untuk **Setujui Pendaftaran**, **Role & Hak Akses**, dan **Parameter Sistem**.
- Nama role dan kombinasi akses dapat dibuat bebas. Pencatatan BTD, Penetapan BDN, Pemasukan Barang Titipan, dan setiap action inventory memiliki permission unik sehingga tugas pengguna dapat dipisahkan secara rinci.
- Pada menu **Role & Hak Akses**, setiap role menampilkan jumlah pengguna. Admin dapat menghapus permanen role yang memiliki **0 pengguna**; role yang masih ditetapkan kepada pengguna ditolak oleh backend dan relasi database.
- Admin dapat mencari, menambah, mengedit, atau menonaktifkan seluruh master dropdown operasional, termasuk satuan, peruntukan, TPS asal, nama TPP, jenis muatan, jenis pengeluaran, dan jenis serah terima. Penonaktifan tidak menghapus nilai historis dari data lama.
- Sesi login berakhir otomatis setelah 30 menit tanpa aktivitas. Aktivitas pengguna disinkronkan antar-tab dan diverifikasi kembali oleh server.
- Administrator dapat menghapus data barang secara permanen dari Inventory. Sebelum penghapusan, sistem menyimpan snapshot barang, proses, dan timeline ke tabel audit database.
- Login admin lokal serta pendaftaran/login email melalui Supabase Auth.
- Mode demo tanpa database untuk mencoba aplikasi dengan data realistis.

Master TPP tujuan/lokasi internal yang digunakan untuk data contoh:

1. TPP Transporindo
2. TPP Multi Sejahtera
3. TPP KBN Marunda
4. TPP Graha Segara

**TPP L4 sengaja tidak dimasukkan** dalam master, dropdown, maupun data contoh.

Daftar **TPS asal** terpisah dari daftar TPP tujuan. Dropdown TPS asal memuat 18 nama TPS yang ditetapkan pada revisi workflow ini.

## Menjalankan secara lokal

Prasyarat: Go 1.23 atau lebih baru. Image deployment menggunakan Go 1.26 dan Alpine 3.24.

```bash
cp .env.example .env
go run ./cmd/server
```

Buka `http://localhost:8080`.

Mode demo akan aktif otomatis jika Supabase belum dikonfigurasi. Jika `ADMIN_USERNAME` dan `ADMIN_PASSWORD` dikosongkan pada development/demo, akun lokal sementara adalah `admin` / `admin-demo-only`. Kredensial demo ditolak otomatis pada `APP_ENV=production`.

Production wajib memakai `SESSION_SECRET` acak minimal 32 karakter, HTTPS, `DEMO_MODE=false`, serta password admin lokal minimal 16 karakter yang bukan password umum.

> Go tidak membaca file `.env` secara otomatis. Saat menjalankan lokal, ekspor variabelnya melalui terminal/IDE atau gunakan pengelola environment pilihan Anda. Aplikasi sengaja tidak memakai dependency eksternal agar build tetap sederhana.

## Menghubungkan Supabase

1. Buka **SQL Editor** pada project Supabase.
2. Untuk database baru, jalankan migration secara berurutan: `001_schema.sql`, `003_livira_upgrade.sql`, `004_workflow_revision.sql`, `005_reporting_item_kind.sql`, `006_history_search_dashboard.sql`, `007_access_approval_parameters.sql`, `008_idle_session_admin_delete.sql`, `009_expand_system_parameters.sql`, `010_capacity_multi_container_dashboard.sql`, `011_container_size_options_ui.sql`, `012_reporting_pagination_multi_goods_pfpd.sql`, `013_titipan_rekonsiliasi_lelang_dashboard.sql`, `014_multi_barang_kondisi_htl_per_item.sql`, `015_document_upload_admin_search_access.sql`, `016_security_performance_hardening.sql`, `017_transfer_lelang_rekonsiliasi_perubahan_data.sql`, `018_reconciliation_tabs_change_audit_reports.sql`, dan `019_btd_dashboard_report_upload_fixes.sql`, lalu `020_btd_bl_date.sql` dan `021_dashboard_titipan_sync.sql`.
3. Jika membutuhkan data contoh, jalankan `002_seed.sql` setelah schema awal dan sebelum digunakan sebagai database operasional. Jangan jalankan file seed pada database produksi yang sudah berisi data.
4. Buka **Project Settings → API** dan salin Project URL, anon key, serta service role key.
5. Atur environment berikut pada server:

```text
DEMO_MODE=false
SUPABASE_URL=https://PROJECT_REF.supabase.co
SUPABASE_ANON_KEY=...
SUPABASE_SERVICE_ROLE_KEY=...
SUPABASE_STORAGE_BUCKET=livira-documents
PUBLIC_BASE_URL=https://alamat-aplikasi-anda
SESSION_SECRET=nilai-acak-panjang
ADMIN_USERNAME=admin-operasional
ADMIN_PASSWORD=password-baru-yang-kuat
```

`SUPABASE_SERVICE_ROLE_KEY` hanya digunakan oleh backend. Jangan menaruhnya di JavaScript, screenshot, atau repository GitHub.

### Konfigurasi OTP email

1. Pastikan **Confirm email** aktif pada **Authentication → Providers → Email**.
2. Buka **Authentication → Email Templates → Confirm signup**.
3. Gunakan variabel `{{ .Token }}` pada isi template agar email menampilkan OTP 6 digit, bukan hanya tautan konfirmasi. Contoh sederhana:

```html
<h2>Kode OTP LIVIRA</h2>
<p>Masukkan kode berikut pada halaman konfirmasi pendaftaran:</p>
<p style="font-size:32px;font-weight:700;letter-spacing:8px">{{ .Token }}</p>
<p>Kode ini bersifat rahasia dan memiliki masa berlaku terbatas.</p>
```

4. Atur **Site URL** sesuai `PUBLIC_BASE_URL`. Untuk penggunaan operasional, konfigurasi custom SMTP direkomendasikan agar pengiriman email lebih andal.

Untuk fitur **Lupa password**, buka **Authentication → Email Templates → Reset Password** dan gunakan `{{ .Token }}` agar email pemulihan menampilkan OTP 6 digit:

```html
<h2>Reset password LIVIRA</h2>
<p>Masukkan kode berikut pada halaman reset password:</p>
<p style="font-size:32px;font-weight:700;letter-spacing:8px">{{ .Token }}</p>
<p>Abaikan email ini jika Anda tidak meminta perubahan password.</p>
```

Jika template Reset Password hanya memakai `{{ .ConfirmationURL }}`, pengguna akan menerima tautan, sedangkan halaman LIVIRA meminta OTP. CAPTCHA login bersifat internal dan tidak memerlukan site key atau secret key layanan CAPTCHA pihak ketiga.

Jika database sebelumnya sudah menggunakan versi SIPANDAI/LIVIRA terdahulu, jangan ulangi schema awal. Jalankan hanya migration yang belum pernah diterapkan. **Jika migration 015 sudah pernah dijalankan, jangan mengulanginya; langsung jalankan [`migrations/016_security_performance_hardening.sql`](migrations/016_security_performance_hardening.sql).** Migration 007 membuat tabel pendaftaran, role, dan parameter dinamis. Migration 008 menambahkan penghapusan inventory yang atomik dan audit snapshot. Migration 009 memperluas master parameter operasional. Migration 010 menambahkan kapasitas YOR/SOR dan dukungan satu penetapan untuk banyak kontainer. Migration 011 menambahkan ukuran 40' HC dan 45' HC. Migration 012 menambahkan identitas unit fisik, multi-uraian per kontainer, paginasi, serta pengelompokan penelitian berdasarkan nomor request. Migration 013 menambahkan barang titipan, rekonsiliasi fisik, pengelompokan hasil lelang berdasarkan ND penjadwalan, pemisahan proses mulai/selesai pada dashboard, dan alur pengeluaran barang musnah sebelum BA Musnah. Migration 014 menambahkan master kondisi barang dan kolom kondisi hasil pencacahan. Migration 015 menambahkan penyimpanan dokumen action, tautan unduh timeline, hak akses pengelolaan kapasitas, normalisasi data lelang tanpa komponen biaya, serta indeks pendukung perhitungan performa dari timeline.

Jika database Anda saat ini sudah sampai migration 006 dan migration 007–010 belum pernah dijalankan, Anda dapat menjalankan satu file [`MIGRATION_GABUNGAN_007_010.sql`](MIGRATION_GABUNGAN_007_010.sql). Jangan menjalankan file gabungan bersamaan dengan migration 007–010 terpisah.

Panduan lengkap tersedia pada [`docs/KONFIGURASI_OTP_DAN_AKSES.md`](docs/KONFIGURASI_OTP_DAN_AKSES.md).

## Upload ke GitHub

1. Ekstrak ZIP.
2. Masuk ke folder hasil ekstrak sampai terlihat `go.mod`, `Dockerfile`, `cmd`, `internal`, dan `migrations`.
3. Upload **seluruh isi folder tersebut** ke repository GitHub, termasuk folder tersembunyi `.github/workflows`.
4. Jangan upload file `.env`. GitHub hanya memerlukan `.env.example` sebagai contoh.

## Deployment

Aplikasi dapat dipasang pada layanan apa pun yang mendukung Docker atau Go:

```bash
docker build -t livira .
docker run --rm -p 8080:8080 --env-file .env livira
```

Health check tersedia di `/healthz`. Server membaca port dari environment `PORT`.

## Struktur proyek

```text
.
├── .github/workflows/ci.yml    # GitHub Actions untuk test, vet, dan build
├── cmd/server/main.go          # entry point server
├── internal/auth/              # Supabase Auth dan signed session cookie
├── internal/config/            # environment configuration
├── internal/domain/            # model dan daftar workflow
├── internal/store/             # memory store dan Supabase/PostgREST store
├── internal/web/               # handler, template, CSS, dan JavaScript
├── migrations/                 # schema dan data contoh Supabase
├── docs/PANDUAN_UPGRADE_LIVIRA.md # langkah upgrade GitHub, Render, dan Supabase
├── docs/KONFIGURASI_OTP_DAN_AKSES.md # setup OTP, approval, role, dan parameter
├── docs/KEAMANAN_SESI_DAN_PENGHAPUSAN.md # timeout 30 menit dan audit penghapusan admin
├── docs/REVISI_034_HAPUS_ROLE_KOSONG.md # penghapusan role tanpa pengguna
├── docs/MAPPING_EXCEL.md       # pemetaan action dari workbook referensi
├── docs/VALIDASI_DATA_EXCEL.md # hasil validasi dan normalisasi data sumber
├── Dockerfile
└── go.mod
```

Data contoh pada repository bersifat fiktif. Data operasional dari workbook tidak ditanam ke source GitHub karena memuat identitas consignee/alamat dan masih memerlukan normalisasi nama TPP; lihat [`docs/VALIDASI_DATA_EXCEL.md`](docs/VALIDASI_DATA_EXCEL.md).

## Aturan proses penting

- Pendaftar harus menyelesaikan OTP email, lalu menunggu persetujuan dan penetapan role oleh admin sebelum dapat login.
- Login wajib melewati CAPTCHA yang valid. Tantangan kedaluwarsa otomatis setelah 5 menit dan kegagalan tetap dihitung oleh pembatasan percobaan login.
- Reset password hanya berlaku untuk akun email Supabase. Password administrator lokal/break-glass tetap diubah melalui environment `ADMIN_PASSWORD` pada server.
- Hak akses role diterapkan pada menu, endpoint, action perubahan data, serta cakupan BTD/BDN/BMMN. Perubahan role/status akun menaikkan versi sesi sehingga sesi aktif dengan hak lama langsung ditolak pada request berikutnya.
- Penghapusan user dari menu Setujui Pendaftaran bersifat permanen untuk profil aplikasi dan identitas Supabase Auth. Jika orang yang sama membutuhkan akses kembali, ia harus mendaftar ulang dan menjalani persetujuan admin.
- Role yang tidak mempunyai pengguna dapat dihapus permanen oleh admin. Role yang masih direferensikan akun tidak dapat dihapus; pindahkan role pengguna terlebih dahulu. Parameter tetap dikelola melalui penonaktifan agar nilai historis data lama tetap utuh.
- Timeout sesi tidak hanya berjalan di browser. Signed session cookie juga menyimpan waktu aktivitas terakhir, sehingga request setelah 30 menit tanpa aktivitas ditolak oleh server.
- Hanya role administrator utama yang dapat menjalankan penghapusan permanen data barang. Tombol Hapus tidak diberikan kepada role kustom.
- Penghapusan permanen menghapus inventory beserta proses dan timeline terkait, tetapi snapshot lengkapnya tetap disimpan dalam `inventory_deletion_audit` untuk kebutuhan pemeriksaan database.
- Inventory hanya memuat barang dengan `is_active = true`.
- Penetapan awal hanya menerima BTD dan BDN.
- Barang baru secara default berstatus **Masih di TPS** dan memakai TPS asal sebagai lokasi.
- Pemindahan ke TPP memperbarui `facility_id`, nama TPP, lokasi, dan status lokasi.
- Jenis barang, kategori BDN, satuan, peruntukan, TPS asal, TPP, jenis muatan, jenis pengeluaran, dan jenis serah terima mengikuti master aktif yang dikelola admin serta divalidasi kembali oleh backend/database.
- Penelitian PFPD hanya dapat dilakukan setelah nomor request tersimpan.
- Asal BMMN tetap disimpan dalam `origin_type`; jenis, nomor, dan tanggal dokumen BCF 1.5/KEP BDN juga disimpan sebelum referensi berubah menjadi dokumen Penetapan BMMN.
- Usulan dan persetujuan peruntukan hanya dapat diterapkan kepada BMMN.
- Lelang, musnah, dan hibah/PSP wajib mereferensikan satu atau lebih `inventory_id` aktif melalui menu Action.
- `current_disposition` mencegah satu barang memiliki beberapa proses aktif.
- Setiap perubahan ditulis ke tabel `events` sehingga timeline tidak bergantung pada status terakhir saja.
- Hasil lelang Laku dapat langsung ditindaklanjuti dengan Pengeluaran Barang; BA Musnah dan BA Serah Terima juga membuka pengeluaran sesuai jenis proses. Pengeluaran mengubah `is_active` menjadi `false`, menutup proses yang masih aktif, dan mempertahankan data pada History, Pencarian Detail, dan Pelaporan.
- Rentang tanggal pada Pelaporan dan Pencarian Detail menggunakan `determination_date`; pilihan **Aktif saja**, **Semua status**, atau **Selesai saja** menggunakan `is_active`.
- Peruntukan BMMN dibatasi pada empat nilai baku: **Lelang, Musnah, Hibah,** dan **PSP**.

## Kapasitas dan jenis muatan

- Kapasitas **YOR** per TPP disimpan dalam TEU dan dapat diedit dari Dashboard oleh pengguna yang memiliki akses Parameter Sistem.
- Kapasitas **SOR** per TPP disimpan dalam m³. Penggunaan YOR/SOR dihitung otomatis dari inventory aktif yang sudah berada di TPP.
- Pencatatan/Penetapan FCL dapat memuat beberapa nomor kontainer beserta ukuran 20', 40', 40' HC, atau 45' HC. Satu kontainer dapat memiliki banyak baris identitas barang, tetapi tetap dihitung satu unit FCL pada dashboard dan kapasitas YOR.
- Penetapan LCL wajib memuat perkiraan volume barang dalam m³.
- Popup Pencatatan BTD, Penetapan BDN, dan Pemasukan Barang Titipan memiliki pilihan **Upload Excel** untuk menyimpan hingga 1.000 baris sekaligus.
- Setiap jenis pemasukan memakai template `.xlsx` yang berbeda, dilengkapi contoh pada baris kedua, referensi pilihan, dan petunjuk pengisian.
- Upload divalidasi secara menyeluruh sebelum disimpan; apabila ada satu baris tidak valid, seluruh upload dibatalkan agar tidak terjadi data parsial.
- Dashboard Lelang, Musnah, dan Hibah/PSP ditampilkan sebagai popup dari Dashboard utama.

Panduan fitur tersedia pada [`docs/REVISI_KAPASITAS_MULTI_KONTAINER_DASHBOARD.md`](docs/REVISI_KAPASITAS_MULTI_KONTAINER_DASHBOARD.md).

## Pengujian

```bash
go test ./...
```

Aplikasi hanya memakai standard library Go, sehingga tidak memerlukan `go get` atau dependency pihak ketiga.

## Revisi antarmuka terbaru

- Sidebar administrator sekarang memiliki area menu yang dapat di-scroll tanpa menghilangkan merek dan status database.
- Lonceng kanan atas membuka pusat notifikasi dinamis dan mengarahkan pengguna ke tindakan yang relevan.
- Identitas pengguna kanan atas dapat diklik untuk membuka menu profil dan logout.
- Popup profil menampilkan email, role, status akun, keamanan sesi, dan hak akses.

Revisi sidebar, notifikasi, profil, upload Excel massal, popup performa, dan ekspor Excel performa tidak membutuhkan tabel agregat baru. Revisi kapasitas dan multi kontainer membutuhkan migration 010, ukuran kontainer tambahan membutuhkan migration 011, multi-uraian dan alur penelitian PFPD membutuhkan migration 012, barang titipan dan rekonsiliasi membutuhkan migration 013, kondisi barang hasil pencacahan membutuhkan migration 014, lampiran dokumen awal dan indeks performa membutuhkan migration 015, private Storage, audit keamanan, session revocation, pencarian trigram, view proses, dan RPC workflow atomik membutuhkan migration 016, pengalihan barang lelang Tidak Laku serta fitur Perubahan Data Barang membutuhkan migration 017, pemisahan tab, audit nilai sebelum dan sesudah, serta pemisahan laporan membutuhkan migration 018, sedangkan nomor BL BTD, detail jumlah hasil pencacahan, metrik dokumen/FCL/LCL, preset Laporan BTD, dan perbaikan upload multi-barang per kontainer membutuhkan migration 019.


### Migration 011 — opsi ukuran peti kemas

Setelah migration multi-kontainer dijalankan, jalankan `migrations/011_container_size_options_ui.sql` untuk menambahkan opsi 40' HC dan 45' HC serta menormalkan data ukuran lama.

### Migration 012 — pelaporan, paginasi, multi-uraian, dan PFPD

Setelah migration 011 berhasil, jalankan `migrations/012_reporting_pagination_multi_goods_pfpd.sql`. Migration ini memungkinkan satu kontainer memiliki beberapa baris uraian tanpa menggandakan perhitungan kapasitas YOR/SOR dan mendukung penelitian per nomor request. Action Pencacahan tidak lagi menampilkan pertanyaan apakah penelitian PFPD diperlukan; hasil pencacahan dapat langsung dipilih pada action Request Penelitian PFPD. Pilihan 10/20/50/100 baris serta scrollbar tabel bagian atas tidak memerlukan kolom database tambahan di luar migration tersebut.

### Migration 013 — barang titipan, rekonsiliasi, dan penyelesaian lelang

Setelah migration 012 berhasil, jalankan `migrations/013_titipan_rekonsiliasi_lelang_dashboard.sql`. Migration ini menambahkan jenis inventory Barang Titipan beserta kategori dan kantor/unit penitip, tabel audit rekonsiliasi, identitas ND penjadwalan pada proses lelang, hak akses baru, serta matriks jenis pengeluaran terbaru. Migration juga menormalkan status lokasi agar langsung menampilkan nama TPS, TPP, atau kantor/unit penitip tanpa awalan `Berada di`, serta melakukan backfill ND penjadwalan dari event lelang yang sudah ada.

### Migration 014 — multi-identitas, kondisi barang, dan HTL per item

Setelah migration 013 berhasil, jalankan `migrations/014_multi_barang_kondisi_htl_per_item.sql`. Migration ini menambahkan kolom `goods_condition`, kelompok parameter admin `goods_condition`, nilai awal Baru/Bekas/Rusak/Segar/Busuk, validasi database berbasis parameter aktif, dan indeks laporan kondisi barang. Migration ini tidak menghapus atau mereset inventory yang sudah ada.

Pada versi ini, action Selesai Lelang diproses sebagai satu bundle berdasarkan ND penjadwalan. Hasil laku/tidak laku dan harga jual ditetapkan per barang, sedangkan nomor/tanggal risalah diterapkan ke seluruh bundle. Barang dengan KEP Musnah juga dapat dikeluarkan dari inventory aktif sebelum BA Musnah, tetapi prosesnya tetap muncul pada menu Musnah sampai tahap BA diselesaikan.

### Migration 015 — lampiran dokumen, akses admin, dan indeks performa

Setelah migration 014 berhasil, jalankan `migrations/015_document_upload_admin_search_access.sql`. Migration ini membuat penyimpanan lampiran PDF/gambar maksimal 8 MB, menghubungkannya dengan event timeline, menambahkan hak akses khusus untuk mengedit kapasitas YOR/SOR, menormalkan data lelang lama agar tidak memiliki komponen biaya, dan menambahkan indeks untuk penghitungan performa berbasis event. File hanya diakses melalui backend aplikasi dengan sesi dan hak akses yang valid. Performa tidak disimpan sebagai angka statis; sistem menghitung ulang dari timeline agar perubahan data tetap tercermin.

### Migration 016 — keamanan dan performa produksi

Bagi database yang sudah menjalankan migration 015, jalankan hanya `migrations/016_security_performance_hardening.sql`. Migration ini:

- memindahkan lampiran baru ke bucket Supabase Storage privat dan tetap dapat membaca lampiran Base64 lama;
- menambahkan checksum SHA-256 dan pemeriksaan kepemilikan dokumen sebelum unduh;
- menambahkan audit log append-only untuk login, inventory, proses, rekonsiliasi, impor/ekspor, unduh, role, parameter, dan tindakan administrasi;
- mencabut sesi lama saat role, izin, verifikasi, atau status persetujuan berubah;
- menjalankan penetapan multi-barang serta perubahan inventory/proses/event per item dalam satu transaksi RPC dengan optimistic locking;
- menghapus pola N+1 pada daftar proses melalui `disposition_details`;
- memakai kolom pencarian trigram terindeks dan ringkasan notifikasi database;
- menerapkan pagination database pada inventory, pencarian detail, dan daftar proses;
- menghitung dashboard proses serta ringkasan laporan melalui RPC agregasi, dan membatasi sumber data performa pada periode terpilih;
- menyinkronkan perubahan label TPP ke nama tampilan inventory tanpa mengubah kode TPP.

Pastikan `SUPABASE_STORAGE_BUCKET=livira-documents` tersedia pada environment deployment. Akun admin lokal bersifat opsional/break-glass: isi `ADMIN_USERNAME` dan `ADMIN_PASSWORD` sekaligus dengan kredensial kuat, atau kosongkan keduanya setelah role admin Supabase siap. Detail upgrade dan batasan operasional tersedia di [`docs/REVISI_KEAMANAN_DAN_PERFORMA_016.md`](docs/REVISI_KEAMANAN_DAN_PERFORMA_016.md).

### Migration 017: pengalihan lelang Tidak Laku dan perubahan data barang

Untuk database yang sudah berada pada migration 016, jalankan `migrations/017_transfer_lelang_rekonsiliasi_perubahan_data.sql` satu kali sebelum deploy source ini. Migration tersebut memungkinkan barang lelang berstatus Tidak Laku dialihkan secara atomik ke Pemusnahan atau Hibah/PSP, menambahkan jenis rekonsiliasi Perubahan Data Barang, dan menyediakan RPC koreksi data yang tetap menjaga ID sistem serta konsistensi status alur.

Untuk project Supabase yang benar-benar baru dan masih kosong, cukup jalankan `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`. Jangan jalankan file setup penuh pada database operasional yang sudah berisi data. Detail fitur tersedia pada [`docs/REVISI_PENGALIHAN_LELANG_DAN_PERUBAHAN_DATA_017.md`](docs/REVISI_PENGALIHAN_LELANG_DAN_PERUBAHAN_DATA_017.md).

## Upload Excel massal

Panduan dan struktur template tersedia pada [`docs/REVISI_UPLOAD_EXCEL_MASSAL.md`](docs/REVISI_UPLOAD_EXCEL_MASSAL.md). Revisi upload tetap memakai struktur sebelumnya. Untuk database yang sudah berada pada migration 018, jangan mengulang migration lama; jalankan migration 019 lalu deploy source terbaru.


### Migration 018: tab terpisah dan audit sebelum-sesudah

Setelah migration 017 berhasil, jalankan `migrations/018_reconciliation_tabs_change_audit_reports.sql`. Migration ini menambahkan alasan perubahan dan rincian audit terstruktur pada setiap rekonsiliasi Perubahan Data Barang. Setiap perubahan pada data utama inventory, dokumen timeline, dan data proses disimpan sebagai pasangan nilai sebelum dan sesudah. Menu Rekonsiliasi serta menu Pelaporan kemudian memisahkan rekap rekonsiliasi fisik dari rekap perubahan data barang, dan ekspor CSV maupun Excel menggunakan sumber audit yang sama dengan tabel aplikasi.

Untuk project Supabase baru yang masih kosong, jalankan hanya `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`.

### Migration 019: dashboard, Laporan BTD, nomor BL, pencacahan, dan upload multi-barang

Setelah migration 018 berhasil, jalankan `migrations/019_btd_dashboard_report_upload_fixes.sql`, kemudian `migrations/020_btd_bl_date.sql`. Migration 019 menambahkan kolom `bl_no` dan `quantity_detail`, sedangkan migration 020 menambahkan `bl_date` untuk tanggal BL wajib pada pencatatan BTD serta memperbarui RPC pembuatan inventory. Dashboard kini menampilkan jumlah dokumen, FCL, dan LCL yang telah dideduplikasi pada setiap kartu inventory. Preset **Laporan BTD** merekap satu baris per dokumen BTD, kemudian mengelompokkan nomor kontainer serta uraian dan jumlah barang per kontainer.

Upload Excel Pencatatan BTD kini mengizinkan nomor kontainer yang sama pada beberapa baris selama seluruh baris berada dalam dokumen BTD yang sama dan metadata kontainernya konsisten. Dengan demikian, satu kontainer dapat berisi beberapa uraian atau jenis barang tanpa menggandakan perhitungan FCL/YOR. Template BTD terbaru juga mewajibkan nomor BL dan tanggal BL.

Untuk project Supabase baru yang masih kosong, jalankan hanya `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`.


### Revisi 022: Laporan BTD lengkap

Preset Laporan BTD serta ekspor CSV dan Excel memuat kolom-kolom esensial, termasuk BL, manifest, TPS asal, TPP, lokasi, ukuran kontainer/volume LCL, rincian barang, pemilik, nilai, dan status. Tidak diperlukan migration baru.


### Migration 020 dan Revisi 023: tanggal BL wajib pada Pencatatan BTD

Setelah migration 019 berhasil, jalankan `migrations/020_btd_bl_date.sql` sebelum deploy source revisi 023. Pencatatan BTD manual dan upload Excel sekarang mewajibkan **Tanggal BL** di samping Nomor BL. Tanggal BL disimpan pada setiap rincian barang, dijaga konsisten untuk baris dengan kontainer/dokumen yang sama, tampil pada detail inventory, dapat dikoreksi melalui Perubahan Data Barang, dan ditambahkan ke preset serta ekspor **Laporan BTD** sehingga laporan sekarang memiliki 18 kolom. Data BTD lama tetap dapat dibaca meskipun tanggal BL sebelumnya belum tersedia.

Untuk project Supabase baru yang masih kosong, jalankan hanya `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`.


### Revisi 024: seluruh laporan dapat diekspor ke Excel

Seluruh preset dan laporan kustom pada menu Pelaporan sekarang menyediakan dua pilihan unduhan: **CSV UTF-8** dan **Excel `.xlsx`**. Pilihan Excel tersedia untuk laporan kustom, barang aktif per TPP, BTD/BDN 60 hari, potensi siap lelang, barang di TPS, BMMN menunggu peruntukan, riwayat selesai, Laporan BTD, Rekap Rekonsiliasi, dan Rekap Perubahan Data Barang. File Excel memakai sumber data dan filter yang sama dengan CSV, dilengkapi judul laporan, header berformat, kolom numerik, pembekuan header, serta filter tabel. Revisi ini tidak memerlukan migration database baru.

### Revisi 025: filter Laporan BTD dan perbaikan kompatibilitas Excel

Preset **Laporan BTD** memiliki filter tanggal BTD, status inventory aktif/selesai, status barang, lokasi TPS/TPP, dan TPP tertentu. Filter yang sama diterapkan pada tabel, CSV, dan Excel. Struktur Open XML pada ekspor `.xlsx` juga diperbaiki agar Microsoft Excel dapat membuka file langsung tanpa dialog pemulihan konten. Revisi ini tidak memerlukan migration database baru. Detail tersedia pada [`docs/REVISI_FILTER_BTD_DAN_PERBAIKAN_EXCEL_025.md`](docs/REVISI_FILTER_BTD_DAN_PERBAIKAN_EXCEL_025.md).


### Migration 021 dan Revisi 027: sinkronisasi dashboard Barang Titipan

Setelah migration 020 berhasil, jalankan `migrations/021_dashboard_titipan_sync.sql`. Migration ini memperbarui RPC dashboard agar kartu Barang Titipan, metrik dokumen/FCL/LCL, rincian per TPP, dan Total inventory aktif berasal dari sumber data aktif yang sama. Total inventory aktif dihitung secara eksplisit sebagai BTD + BDN + BMMN + Barang Titipan. Untuk project Supabase baru yang kosong, jalankan hanya `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`.

## Revisi 029 — Rebranding LIVIRA dan bongkar/muat per kontainer

Versi ini menggunakan identitas **LIVIRA — Layanan Inventori, Verifikasi, Integrasi, Rekonsiliasi, dan Analitik**. Cakupan dashboard kini mendukung **Masih di TPS**, **Seluruh TPP**, dan masing-masing TPP, dengan seluruh KPI dihitung ulang sesuai pilihan. Target bongkar/pindah FCL dideduplikasi per kontainer; seluruh uraian dalam kontainer akan ditampilkan dan wajib dialokasikan.

- Database operasional: jalankan `migrations/029_livira_rebrand_dashboard_scope_container_target.sql`.
- Database baru dan kosong: jalankan hanya `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`.
- Panduan rinci: `docs/REVISI_029_LIVIRA_DASHBOARD_BONGKAR_MUAT.md`.

## Hotfix 030 setelah rebranding LIVIRA

Jika deployment menampilkan **“Data belum dapat dimuat. Periksa konfigurasi database dan coba kembali.”** setelah migration 029, jalankan `migrations/030_fix_rebrand_function_body_and_postgrest_cache.sql`, lalu redeploy/restart layanan. Hotfix memperbaiki referensi function lama di dalam body RPC dan memuat ulang schema cache PostgREST tanpa mengubah data inventory.


## Revisi 031 — cakupan kantor, permission inventory granular, dan bongkar/muat lintas status

- Opsi paling atas pada **Cakupan inventory** adalah **Seluruh cakupan Kantor Tanjung Priok**, yang menjumlahkan seluruh inventory aktif di TPS dan TPP.
- Opsi Masih di TPS, Seluruh TPP, dan masing-masing TPP hanya memengaruhi kartu KPI. **Detail per TPP** selalu dihitung dari barang yang benar-benar berada di TPP.
- Permission lama **Kelola inventory** dipecah menjadi permission input awal BTD, BDN, Barang Titipan, serta permission terpisah untuk setiap action sampai Pengeluaran Barang.
- Bongkar/Muat Kontainer dapat digunakan pada seluruh inventory aktif, termasuk barang yang sedang lelang, musnah, hibah, atau PSP. Barang yang sedang/sudah berproses hanya boleh dipindahkan ke satu tujuan per uraian agar relasi proses tetap konsisten.
- Database operasional yang sudah sampai hotfix 030: jalankan `migrations/031_dashboard_office_scope_granular_inventory_access.sql`, kemudian deploy source terbaru.
- Database baru dan kosong: jalankan hanya `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`.
- Perubahan permission role akan menaikkan versi sesi pengguna terkait; pengguna non-admin mungkin perlu login kembali.
- Panduan rinci: `docs/REVISI_031_CAKUPAN_KANTOR_HAK_AKSES_BONGKAR_MUAT.md`.

## Revisi 032 — status bongkar/muat dan template upload

- Pada action **Bongkar/Muat Kontainer**, validasi form diperbarui tanpa memindahkan ulang elemen DOM ketika pengguna mengetik, sehingga fokus/kursor tetap berada pada kotak isian.
- Action bongkar/muat hanya mengubah penempatan fisik, nomor/ukuran kontainer, kuantitas, volume, dan perhitungan okupansi. **Status inventory tidak berubah** dan `pindah_bongkar_kontainer` hanya disimpan sebagai event timeline/audit.
- Database operasional yang sudah sampai migration 031: jalankan `migrations/032_bongkar_muat_preserve_inventory_status.sql`, lalu deploy source terbaru.
- Database baru dan kosong: jalankan hanya `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`.
- Template BTD hanya memiliki satu baris contoh pada baris 2. Template BTD dan BDN menggunakan contoh nomor kontainer tanpa spasi/tanda hubung, misalnya `ABCD1234567`.

## Revisi 034 — hapus role tanpa pengguna

- Setiap kartu pada **Admin → Role & Hak Akses** menampilkan jumlah pengguna yang memakai role tersebut.
- Tombol **Hapus role** hanya ditampilkan ketika jumlah pengguna adalah 0 dan dilindungi konfirmasi serta CSRF.
- Backend memvalidasi ulang kondisi saat permintaan dikirim. Foreign key `app_users.role_id` dengan `ON DELETE RESTRICT` menjadi perlindungan terakhir terhadap kondisi balapan atau pemanggilan endpoint langsung.
- Penghapusan role dicatat pada audit log dengan hasil sukses atau gagal.
- Tidak ada migration tambahan. Database baru tetap menggunakan `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`; database yang sudah berjalan cukup dideploy dengan source terbaru.
