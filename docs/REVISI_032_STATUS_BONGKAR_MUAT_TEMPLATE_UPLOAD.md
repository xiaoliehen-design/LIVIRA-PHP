# Revisi 032 — Status Bongkar/Muat dan Template Upload

## Perubahan

1. Input action **Bongkar/Muat Kontainer** tidak lagi kehilangan fokus setelah setiap karakter. Pembaruan validitas tombol simpan dilakukan tanpa reorder atau sinkronisasi ulang elemen editor selama pengguna mengetik.
2. Bongkar/muat tidak menjadi status inventory. Status dan proses barang sebelum action dipertahankan pada sumber maupun hasil pembagian barang. Event `pindah_bongkar_kontainer` tetap dicatat pada timeline untuk audit.
3. Migration 032 memulihkan data lama yang sempat memiliki status `pindah_bongkar_kontainer` menggunakan event terakhir sebelum bongkar/muat.
4. Template upload BTD hanya menyediakan satu baris contoh pada baris 2.
5. Contoh nomor kontainer pada template BTD dan BDN memakai format kompak tanpa spasi/tanda hubung, misalnya `ABCD1234567`.

## Implementasi database

- Database operasional yang sudah menjalankan migration 031: jalankan `migrations/032_bongkar_muat_preserve_inventory_status.sql`.
- Database baru dan kosong: jalankan `migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql`.
- Setelah migration berhasil, deploy source terbaru.
