# Revisi Kapasitas YOR/SOR, Multi Kontainer, dan Dashboard Proses

## 1. Kapasitas YOR dan SOR per TPP

Pada Dashboard, pengguna dengan hak akses **Parameter Sistem** dapat menekan tombol **Edit kapasitas** dan memilih TPP yang akan diperbarui.

- **Kapasitas YOR** dicatat dalam TEU atau ekuivalen peti kemas 20 kaki.
- **Kapasitas SOR** dicatat dalam meter kubik (`m³`).
- Penggunaan YOR dihitung otomatis dari inventory FCL aktif yang sudah berada di TPP:
  - kontainer 20 kaki = 1 TEU;
  - kontainer 40 kaki = 2 TEU;
  - kontainer 45 kaki = 2,25 TEU.
- Penggunaan SOR dihitung otomatis dari total perkiraan volume inventory LCL aktif yang sudah berada di TPP.

Nilai pemakaian tidak perlu diedit secara manual karena mengikuti barang aktif dan berubah ketika barang dipindahkan, dikeluarkan, atau dihapus.

## 2. Penetapan FCL dengan banyak kontainer

Pada form Pencatatan BTD atau Penetapan BDN:

1. Pilih jenis muatan **FCL**.
2. Pilih ukuran kontainer 20', 40', 40' HC, atau 45' HC.
3. Masukkan nomor kontainer.
4. Tekan **Enter** atau tombol **Tambah kontainer**.
5. Ulangi untuk kontainer berikutnya.

Satu nomor BCF 1.5 atau satu KEP BDN dapat menghasilkan beberapa baris inventory. Setiap baris memiliki nomor referensi unik, tetapi tetap menyimpan nomor penetapan yang sama. Sistem mencegah nomor kontainer aktif tercatat dua kali.

## 3. Penetapan LCL

Jika jenis muatan **LCL** dipilih, form menampilkan isian **Perkiraan volume barang (m³)**. Nilai ini wajib lebih besar dari nol dan digunakan untuk menghitung keterisian SOR.

## 4. Dashboard Lelang, Musnah, dan Hibah/PSP

Grafik proses tidak lagi memenuhi halaman masing-masing proses. Ringkasan proses berada di Dashboard utama:

- klik informasi **Lelang** untuk membuka popup dashboard lelang;
- klik informasi **Musnah** untuk membuka popup dashboard pemusnahan;
- klik informasi **Hibah/PSP** untuk membuka popup dashboard hibah dan PSP.

Halaman Lelang, Musnah, dan Hibah tetap difokuskan pada daftar proses, workflow, Action, dan History.

## Migration

Untuk database yang sudah sampai migration 009, jalankan:

```text
migrations/010_capacity_multi_container_dashboard.sql
```

Jika database saat ini baru sampai migration 006 dan perubahan 007–009 belum pernah diterapkan, gunakan file gabungan 007–010 yang disertakan bersama paket rilis.
