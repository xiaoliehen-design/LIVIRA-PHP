# Timeout Sesi dan Penghapusan Data Barang

## Timeout tanpa aktivitas

LIVIRA menggunakan batas tidak aktif selama **30 menit**.

- Browser mencatat aktivitas seperti klik, keyboard, scroll, sentuhan, dan pergerakan pointer.
- Aktivitas disinkronkan antar-tab pada browser yang sama.
- Selama pengguna aktif, browser mengirim heartbeat berkala ke backend.
- Backend menyimpan `last_activity` dalam signed session cookie dan menolak sesi yang melewati 30 menit tanpa aktivitas.
- Saat waktu habis, cookie dibersihkan dan pengguna diarahkan ke halaman login dengan pemberitahuan bahwa sesi berakhir otomatis.

Batas ini tidak memerlukan perubahan tabel database dan sudah aktif ketika source Go terbaru digunakan.

## Penghapusan barang oleh administrator

Tombol **Hapus** hanya muncul untuk akun dengan role internal `admin`. Role kustom tetap tidak dapat mengakses endpoint penghapusan walaupun memiliki izin kelola Inventory.

Sebelum data dihapus, fungsi database `admin_delete_inventory` menyimpan:

- snapshot baris `inventory_items`;
- seluruh proses pada `dispositions`;
- seluruh timestamp dan timeline pada `events`;
- nama administrator serta waktu penghapusan.

Snapshot tersimpan di tabel `inventory_deletion_audit`. Setelah audit tersimpan dalam transaksi yang sama, sistem menghapus proses, timeline, dan barang sehingga tidak lagi muncul pada menu operasional maupun laporan.

Untuk mengaktifkan fungsi ini, jalankan:

```text
migrations/008_idle_session_admin_delete.sql
```

Migration tersebut harus dijalankan setelah migration 007.
