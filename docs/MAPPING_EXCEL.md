# Pemetaan Workbook BCP ke Workflow LIVIRA

Dokumen ini mencatat bagaimana struktur `BCP BTD.xlsx`, `BCP BDN.xlsx`, dan `BCP BMMN.xlsx` diterjemahkan ke aplikasi.

## Data dasar inventory

| Kelompok Excel | Field aplikasi |
|---|---|
| BC 1.1/Manifest | `manifest_no`, `manifest_date`, `manifest_position` |
| Pencatatan BTD/Penetapan BDN | `determination_no`, `determination_date`, `item_type` |
| Uraian/Jumlah/Satuan/Lokasi | `description`, `quantity`, `unit`, `location` |
| Shipper/Consignee/Pemilik | `owner_name`, `owner_address` |
| TPS/Asal barang | `origin_warehouse` |
| TPP | `facility_id`, `facility_name` |
| Data kontainer | `load_type`, `container_no` |
| Status barang | `status_code`, `status_label` dan tabel `events` |

## Action BTD dan BDN

Action berikut dipindahkan dari kolom-kolom lebar Excel menjadi form tindak lanjut bertahap:

1. Pencatatan BTD atau penetapan BDN sebagai input awal inventory.
2. Pemindahan BTD/BDN — nomor dan tanggal ST/SPRIN/BA.
3. Pemberitahuan BTD/BDN — nomor dan tanggal surat.
4. Pencacahan — nomor dan tanggal BA cacah.
5. Request Penelitian PFPD — nomor/tanggal dokumen permintaan penelitian.
6. Penelitian PFPD — kode HS, status/keterangan lartas, dan nilai barang berdasarkan request.
7. Penetapan BMMN — nomor dan tanggal SKEP BMMN; action ini mengubah `item_type` menjadi `BMMN` dan mempertahankan `origin_type`.
8. Usulan peruntukan BMMN — jenis peruntukan, nomor dokumen usulan, dan tanggal; hanya tersedia untuk BMMN.
9. Persetujuan peruntukan BMMN — jenis peruntukan, nomor dokumen persetujuan, dan tanggal; hanya tersedia untuk BMMN.
10. Pengeluaran barang — nomor/tanggal dokumen dan pilihan jenis pengeluaran khusus BTD, BDN, atau BMMN.

Lelang, pemusnahan, dan hibah/PSP tetap dimulai dengan mencari kontainer/barang dari inventory aktif.

## Action lelang

Pemetaan action Excel mencakup:

- Penerbitan KEP Lelang sekaligus memilih barang dari inventory;
- Penerbitan KEP Harga Terendah Lelang dengan nilai HTL terpisah dari nilai PFPD;
- Penjadwalan Lelang untuk tanggal tunggal atau rentang tanggal;
- Selesai Lelang dengan hasil laku/tidak laku dan nilai terjual;
- Lelang Penyesuaian khusus barang tidak laku, tanpa tahapan KEP HTL baru;
- Alokasi Hasil Lelang dengan KEP dan keterangan tujuan alokasi.

Nomor putaran tersimpan pada `dispositions.round`; validasi status mencegah tahapan dilewati.

## Action pemusnahan

- nomor/tanggal KEP Musnah dan biaya musnah;
- nomor/tanggal BA Musnah dan biaya aktual.

## Action hibah/PSP

- jenis Hibah atau PSP;
- nomor dan tanggal BA Serah Terima.

## Perubahan utama dari Excel

Excel menyimpan status sebagai formula yang membaca pasangan nomor/tanggal dokumen. Pada aplikasi, setiap action membuat satu record `events` dengan timestamp server. Status terakhir tetap disimpan pada inventory untuk pencarian cepat, sedangkan riwayat lengkap tidak pernah ditimpa.
