# Revisi 020: Rekonsiliasi dan audit perubahan data barang

## Perubahan antarmuka

Menu Rekonsiliasi sekarang memiliki dua tombol tindakan:

1. **Rekonsiliasi** untuk mencatat barang yang tercatat tetapi tidak ditemukan, atau barang yang ditemukan tetapi belum tercatat.
2. **Perubahan data barang** untuk memperbarui data bisnis, dokumen timeline, dan data proses.

Halaman utama juga memiliki dua tab terpisah:

- **Rekonsiliasi** hanya menampilkan hasil perbandingan catatan aplikasi dan kondisi fisik.
- **Perubahan data barang** menampilkan data yang berubah, nilai sebelum, nilai sesudah, alasan perubahan, waktu, dan petugas.

## Audit perubahan

Migration 018 menambahkan kolom `correction_reason` dan `change_details` pada tabel `reconciliations`. Rincian perubahan disimpan secara atomik bersama pembaruan data. Sistem menolak penyimpanan apabila tidak ada nilai yang benar-benar berubah.

Audit mencakup:

- data utama inventory;
- nomor dan tanggal dokumen pada timeline;
- nilai dan data proses lelang, pemusnahan, serta hibah/PSP.

## Pelaporan

Menu Pelaporan menyediakan dua preset yang berbeda:

- **Rekap rekonsiliasi**;
- **Rekap perubahan data barang**.

Ekspor Rekap perubahan data barang menghasilkan satu baris pada CSV maupun Excel untuk setiap data yang berubah. Isi ekspor berasal dari `change_details` yang sama dengan tabel pada aplikasi sehingga nilai sebelum dan sesudah tetap konsisten.

## Instalasi

Database yang sudah menggunakan migration 017 harus menjalankan:

```text
migrations/018_reconciliation_tabs_change_audit_reports.sql
```

Database baru yang masih kosong cukup menjalankan:

```text
migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql
```
