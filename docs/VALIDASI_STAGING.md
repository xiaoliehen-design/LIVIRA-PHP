# Checklist Validasi Staging LIVIRA PHP

Gunakan akun dan database staging/salinan bila memungkinkan.

## Autentikasi dan keamanan

- Login admin berhasil.
- Login user Supabase berhasil.
- CAPTCHA salah ditolak; CAPTCHA benar hanya dapat dipakai satu kali.
- OTP pendaftaran dan kirim ulang OTP berjalan.
- Lupa password dan pembuatan password baru berjalan.
- Sesi logout setelah 30 menit tidak aktif.
- CSRF menolak form tanpa token valid.
- User non-admin tidak dapat membuka URL data tipe/proses yang tidak diizinkan.

## Admin

- Setujui, tolak, ubah role, dan hapus user.
- Buat/edit/nonaktifkan role.
- Role tanpa user dapat dihapus.
- Role yang masih digunakan tidak dapat dihapus.
- Parameter dan data TPP dapat ditambah/edit/nonaktifkan.

## Inventory

- Input BTD/BDN/barang titipan FCL dan LCL.
- FCL multi-kontainer dan multi-rincian barang.
- Upload Excel massal serta rollback ketika satu baris tidak valid.
- Pemindahan, pencacahan, request/penelitian PFPD, BMMN, peruntukan, dan pengeluaran.
- Pada pencacahan FCL, tambahkan beberapa uraian baru lalu pastikan setiap uraian menjadi baris inventory tersendiri.
- Pastikan seluruh uraian lama tetap tercantum, status berubah menjadi Pencacahan, dan hitungan FCL/YOR tidak bertambah karena uraian baru.
- Bongkar/muat tidak mengubah status proses barang sebelumnya.
- Nomor kontainer tersimpan dalam format `ABCD1234567`.

## Penyelesaian

- Lelang: KEP, HTL per barang, jadwal, laku/tidak laku, alokasi.
- Pemusnahan: KEP dan BA Musnah.
- Hibah/PSP: dokumen dan BA Serah Terima.
- Proses selesai berpindah ke history.

## Rekonsiliasi dan pelaporan

- Rekonsiliasi fisik menambah/mengeluarkan inventory sesuai pilihan.
- Perubahan data menyimpan nilai sebelum/sesudah dan alasan.
- Laporan rekonsiliasi terpisah dari laporan perubahan data.
- Preset laporan, filter, pagination, CSV, XLS, dan XLSX konsisten.
- Laporan BTD terdeduplikasi per dokumen dan kontainer.
- Performa terdeduplikasi per nomor/tanggal dokumen penyelesaian.

## Storage

- Upload PDF/gambar maksimal 8 MB.
- File tersimpan pada bucket yang benar.
- Download berhasil dan hash/ukuran file konsisten.
- Pengguna tanpa izin tidak dapat mengunduh file dengan menebak ID.

## Regresi bongkar/muat v1.0.7

- Pilih satu kontainer FCL, ubah ke nomor kontainer tujuan yang berbeda, lalu simpan.
- Pastikan tidak muncul error `22P02` atau UUID kosong.
- Pastikan seluruh uraian yang dipilih berpindah sesuai alokasi dan jumlah alokasi sama dengan jumlah sumber.
- Uji muat LCL ke FCL dan bongkar FCL ke LCL.
- Pastikan status proses barang sebelumnya tidak berubah akibat bongkar/muat.
