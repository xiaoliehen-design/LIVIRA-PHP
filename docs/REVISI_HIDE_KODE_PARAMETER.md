# Revisi UI Parameter Sistem — Kolom Kode Disembunyikan

## Perubahan

- Kolom **Kode** tidak lagi ditampilkan pada tabel **Daftar parameter sistem**.
- Nilai kode teknis tetap disimpan secara internal dan tidak diubah ketika label diedit.
- Placeholder pencarian disederhanakan menjadi pencarian berdasarkan kelompok, label, atau cakupan.
- Informasi kode teknis tidak lagi ditampilkan pada panel edit parameter.

## Dampak database

Perubahan hide kode parameter tidak mengubah struktur database. Untuk deployment paket final ini, migration 015 tidak dijalankan ulang; gunakan migration 016 sesuai panduan upgrade.
