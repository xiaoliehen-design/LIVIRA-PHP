# Revisi 028 — Filter Dashboard dan Pindah/Bongkar Kontainer

## 1. Filter cakupan KPI dashboard

Di sebelah tanggal dashboard tersedia dropdown **Cakupan inventory** dengan pilihan:

- **Customs**: seluruh barang aktif dalam pengawasan, baik masih di TPS maupun sudah berada di TPP.
- **Seluruh TPP**: hanya barang aktif yang sudah berada di salah satu TPP.
- **TPP tertentu**: hanya barang aktif pada TPP yang dipilih, misalnya TPP Transporindo.

Pilihan tersebut memperbarui angka dan ringkasan dokumen/FCL/LCL pada kartu:

- Total inventory aktif
- BTD
- BDN
- BMMN
- Barang Titipan

Filter kapasitas YOR/SOR per TPP yang sudah ada tetap terpisah agar pengguna masih dapat membuka rincian kapasitas satu TPP tanpa mengubah makna cakupan KPI.

## 2. Action Pindah/Bongkar Kontainer

Menu **Inventory → Action → Pindah/Bongkar Kontainer** memproses satu uraian barang dalam satu transaksi. Pengguna dapat:

- memindahkan seluruh kuantitas ke kontainer lain;
- mempertahankan sebagian kuantitas di kontainer asal dan memindahkan sisanya;
- membagi barang ke beberapa kontainer;
- membongkar sebagian atau seluruh barang menjadi LCL di gudang;
- menggabungkan tujuan FCL dan LCL dalam satu penyimpanan.

Contoh sumber 10 unit:

- 3 unit → kontainer B;
- 4 unit → kontainer C;
- 3 unit → LCL.

### Aturan integritas

- Total kuantitas seluruh tujuan wajib sama dengan kuantitas sumber.
- Kuantitas tidak boleh nol atau negatif.
- Nomor kontainer tujuan dinormalisasi menjadi format `ABCD 123456-7`.
- Satu nomor kontainer tujuan tidak boleh ditulis dua kali dalam transaksi yang sama.
- Tujuan FCL wajib mempunyai ukuran 20', 40', 40' HC, atau 45' HC.
- Tujuan LCL wajib mempunyai perkiraan volume dalam m³.
- Penyimpanan tanpa perubahan tujuan ditolak.
- Barang yang sudah berada dalam proses disposisi atau sudah menyelesaikan proses tidak dapat dipindah melalui action ini.

### Konsistensi data

- Baris sumber dipertahankan untuk tujuan pertama; tujuan tambahan dibuat sebagai baris inventory baru.
- Nilai barang dibagi proporsional berdasarkan kuantitas dan total nilai tetap sama persis dengan nilai sumber.
- Riwayat/timeline sebelumnya disalin ke baris hasil pembagian, lalu event pindah/bongkar ditambahkan satu kali pada setiap hasil.
- `physical_unit_id` dan `occupancy_primary` diseimbangkan ulang sehingga satu kontainer/LCL hanya dihitung satu kali dalam YOR/SOR.
- Proses Supabase dijalankan melalui satu fungsi RPC transaksional sehingga seluruh pembagian berhasil bersama-sama atau seluruhnya dibatalkan.

## 3. Implementasi database

Untuk database yang sudah berjalan, jalankan:

`migrations/028_dashboard_scope_pindah_bongkar_kontainer.sql`

Untuk database baru, jalankan file setup gabungan:

`migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`

File setup gabungan tersebut sudah memuat migration 028 pada bagian akhir meskipun nama historis filenya tetap dipertahankan agar prosedur deployment lama tidak berubah.

## 4. Verifikasi

Kode telah diperiksa dengan:

- `gofmt` untuk seluruh file Go yang berubah;
- `go test ./...`;
- `node --check internal/web/static/app.js`.
