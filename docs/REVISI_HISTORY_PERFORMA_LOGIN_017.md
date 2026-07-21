# Revisi 017: History proses, performa dashboard, dan halaman login

## Perubahan

1. Halaman utama Lelang tidak lagi menampilkan barang dengan hasil `laku` atau yang sudah masuk tahap `alokasi_hasil_lelang`.
2. Riwayat Lelang menampilkan barang dengan hasil `laku` dan `alokasi_hasil_lelang`, termasuk saat inventory masih aktif.
3. Halaman utama Pemusnahan tidak lagi menampilkan proses dengan status `ba_musnah`.
4. Riwayat Pemusnahan menampilkan proses yang sudah selesai dengan status `ba_musnah`, termasuk apabila pengeluaran fisik dilakukan pada tahap yang berbeda.
5. Kartu Performa Kinerja pada dashboard langsung menghitung tahun kalender berjalan saat dashboard pertama kali dibuka.
6. Popup performa tetap dapat memakai rentang tanggal khusus. Filter khusus hanya mengatur periode dan membuka popup, bukan memicu penghitungan pertama.
7. Halaman login, pendaftaran, dan OTP diperbarui dengan tampilan yang lebih modern dan responsif.
8. Kalimat pada halaman autentikasi tidak menggunakan tanda pisah panjang.

## Database

Perubahan ini tidak memerlukan migration SQL baru. Sistem memakai kolom dan view yang sudah tersedia pada versi database sebelumnya.

## File utama yang berubah

- `internal/domain/models.go`
- `internal/store/store.go`
- `internal/store/memory.go`
- `internal/store/supabase.go`
- `internal/web/server.go`
- `internal/web/templates/process.html`
- `internal/web/templates/auth.html`
- `internal/web/static/app.css`

## Validasi

- `go test ./...`
- `go vet ./...`
- `go build ./...`

Semua pemeriksaan berhasil dijalankan.
