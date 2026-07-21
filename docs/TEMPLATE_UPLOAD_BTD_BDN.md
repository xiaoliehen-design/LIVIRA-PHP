# Template Upload BTD dan BDN

## Lokasi unduhan

Template publik tersedia pada:

- `/templates/template_upload_btd.xlsx`
- `/templates/template_upload_bdn.xlsx`

Path kompatibilitas lama tetap tersedia pada `/assets/templates/`.

## Aturan pengisian

- Gunakan file `.xlsx` dengan ukuran maksimal 6 MB.
- Maksimal 1.000 baris data, tidak termasuk header.
- Baris 2 merupakan satu-satunya contoh dan harus dihapus atau ditimpa sebelum upload.
- Satu baris mewakili satu identitas/uraian barang.
- Nomor kontainer FCL ditulis tanpa spasi atau tanda hubung, contoh `ABCD1234567`.
- Beberapa uraian dalam satu kontainer FCL ditulis pada beberapa baris dengan nomor kontainer dan data dokumen yang sama.
- Untuk FCL, nomor serta ukuran kontainer wajib diisi; volume LCL dikosongkan.
- Untuk LCL, volume wajib diisi; nomor serta ukuran kontainer dikosongkan.
- Bila barang sudah berada di TPP, pilih `Ya` dan isi nama TPP aktif sesuai dropdown.
- Seluruh baris divalidasi terlebih dahulu. Bila satu baris tidak valid, tidak ada data yang disimpan.

## Verifikasi setelah deploy

1. Buka halaman Inventory → Pencatatan BTD atau Penetapan BDN → Upload Excel.
2. Klik tombol unduh template dan pastikan browser menerima file `.xlsx`.
3. Isi atau pertahankan satu baris contoh untuk pengujian pada data staging.
4. Upload dan pastikan notifikasi sukses muncul serta jumlah inventory bertambah.
