# Revisi Pengalihan Lelang dan Perubahan Data Barang 017

## Perubahan alur lelang Tidak Laku

Barang yang telah selesai dilelang dengan hasil **Tidak Laku** sekarang dapat dipilih langsung dari Action pada menu **Pemusnahan** atau **Hibah/PSP**.

Saat action awal disimpan, sistem menjalankan satu transaksi yang:

1. menutup proses lelang Tidak Laku;
2. memberi status riwayat **Dialihkan ke pemusnahan** atau **Dialihkan ke hibah/PSP**;
3. mencatat pengalihan pada timeline barang;
4. membuat proses tujuan yang baru; dan
5. menghapus barang tersebut dari daftar utama Lelang.

Jejak proses lama tetap tersedia pada **History Lelang** untuk kebutuhan audit.

## Rekonsiliasi Perubahan Data Barang

Menu **Catat rekonsiliasi** memiliki opsi ketiga, yaitu **Perubahan data barang**. Petugas dapat memperbarui:

- identitas inventory, nomor referensi, nomor penetapan atau BCF, manifest, jenis dan kategori;
- uraian, jumlah, satuan, nilai, kondisi, pemilik, lokasi, kontainer, ukuran, dan volume;
- data penelitian PFPD, HS, lartas, dokumen asal, usulan, persetujuan, dan pengeluaran;
- nomor dan tanggal surat, ND, KEP, BA, risalah, label, serta catatan pada timeline;
- nilai HTL, nilai terjual, ND jadwal, penerima, hasil lelang, biaya musnah, dan jenis Hibah/PSP.

ID sistem, status alur, status aktif, dan jenis proses tidak dapat diedit melalui form ini. Pembatasan tersebut mencegah data menjadi tidak konsisten dengan workflow.

Sebelum menyimpan, petugas wajib memilih salah satu alasan:

- **Kesalahan input**; atau
- **Error pada saat pengisian awal**.

Setiap koreksi otomatis menghasilkan catatan rekonsiliasi, event timeline **Perubahan data barang**, dan audit backend. Lampiran pendukung bersifat opsional.

## Deployment

### Database yang sudah menjalankan migration 016

Jalankan satu kali:

```text
migrations/017_transfer_lelang_rekonsiliasi_perubahan_data.sql
```

Setelah SQL berhasil, deploy source terbaru ke GitHub/Render.

### Database baru tanpa data

Jalankan:

```text
migrations/01_SETUP_DATABASE_BARU_KOSONG_LIVIRA_001_032.sql
```

File setup penuh tidak boleh dijalankan pada database operasional yang sudah berisi data.
