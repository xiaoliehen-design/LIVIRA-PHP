# Revisi Multi Barang, Kondisi Barang, dan HTL Per Item

## Migration yang diperlukan

Database yang sudah menjalankan migration 013 hanya perlu menjalankan:

```text
migrations/014_multi_barang_kondisi_htl_per_item.sql
```

Migration 014 tidak menghapus atau mereset inventory. Migration menambahkan kolom `goods_condition`, parameter kondisi barang, validasi database, dan indeks pelaporan.

## Perubahan alur input awal

### FCL

Satu dokumen dapat berisi banyak kontainer. Setiap kontainer dapat memiliki banyak identitas barang. Setiap identitas barang disimpan sebagai satu baris inventory, tetapi seluruh baris dalam kontainer yang sama memakai `physical_unit_id` yang sama. Hanya satu baris ditandai sebagai `occupancy_primary`, sehingga kapasitas YOR tetap menghitung satu kontainer.

### LCL

Satu dokumen LCL dapat memiliki banyak identitas barang. Setiap identitas disimpan sebagai satu baris inventory, sedangkan perhitungan SOR tetap memakai satu unit fisik dan satu nilai volume pengiriman.

## Perubahan pencacahan

- Pencarian FCL dilakukan berdasarkan nomor kontainer.
- Pencarian LCL dilakukan berdasarkan uraian barang.
- Beberapa target dapat dipilih sekaligus dan masing-masing mempunyai form hasil pencacahan sendiri.
- Saat kontainer FCL dipilih, seluruh uraian yang sudah tersimpan ditampilkan.
- Uraian baru dapat ditambahkan apabila ditemukan barang yang belum tercakup pada penetapan awal.
- Setiap uraian mempunyai kondisi barang: Baru, Bekas, Rusak, Segar, atau Busuk.
- Uraian baru hasil pencacahan disimpan sebagai baris inventory baru tanpa menambah pemakaian kapasitas YOR.

## Parameter dan pelaporan

Administrator dapat membuka menu **Admin → Parameter sistem → Kondisi barang**. Nilai awal yang disediakan adalah Baru, Bekas, Rusak, Segar, dan Busuk. Parameter aktif ditampilkan pada form pencacahan dan filter menu laporan.

## Perubahan KEP Harga Terendah Lelang

Pada action **Penerbitan KEP Harga Terendah Lelang**, setiap barang yang dipilih memperoleh input nilai HTL masing-masing. Nilai tersebut hanya memperbarui proses lelang barang bersangkutan dan tidak menimpa nilai barang hasil penelitian PFPD.

## Urutan deployment

1. Cadangkan database atau lakukan deployment awal di environment staging.
2. Jalankan `migrations/014_multi_barang_kondisi_htl_per_item.sql` melalui Supabase SQL Editor.
3. Deploy source code versi terbaru.
4. Lakukan hard refresh browser dengan `Ctrl + F5`.
5. Uji input FCL multi-kontainer, FCL multi-uraian, LCL multi-uraian, pencacahan multi-target, penambahan uraian hasil pencacahan, filter kondisi barang, dan HTL per barang.

## Validasi teknis

Versi ini telah melewati:

```text
node --check internal/web/static/app.js
go test ./... -count=1
go vet ./...
go build ./cmd/server
```
