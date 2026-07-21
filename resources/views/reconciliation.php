<section class="page-intro page-intro-actions reconciliation-page-intro">
  <div><p><?= tpl_escape(tpl_get($ctx, 'Subtitle')) ?></p></div>
  <?php if (tpl_truthy(tpl_get($ctx, 'CanManage'))): ?><div class="intro-actions reconciliation-header-actions">
    <button class="button secondary" type="button" data-open-reconciliation data-reconciliation-mode="reconciliation"><svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>Rekonsiliasi</button>
    <button class="button primary" type="button" data-open-reconciliation data-reconciliation-mode="data_correction"><svg viewBox="0 0 24 24"><path d="M4 20h4l10-10-4-4L4 16v4ZM13 7l4 4"/></svg>Perubahan data barang</button>
  </div><?php endif; ?>
</section>

<nav class="reconciliation-tabs" aria-label="Jenis catatan rekonsiliasi">
  <a class="<?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ReconciliationTab'), 'rekonsiliasi'))): ?>active<?php endif; ?>" href="/rekonsiliasi?tab=rekonsiliasi"><span>Rekonsiliasi</span><strong><?= tpl_escape(tpl_len(tpl_get($ctx, 'Reconciliations'))) ?></strong><small>Selisih catatan dan kondisi fisik</small></a>
  <a class="<?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ReconciliationTab'), 'perubahan-data'))): ?>active<?php endif; ?>" href="/rekonsiliasi?tab=perubahan-data"><span>Perubahan data barang</span><strong><?= tpl_escape(tpl_len(tpl_get($ctx, 'DataCorrections'))) ?></strong><small>Audit nilai sebelum dan sesudah</small></a>
</nav>

<?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ReconciliationTab'), 'perubahan-data'))): ?>
<section class="panel table-panel">
  <div class="table-meta"><p><strong><?= tpl_escape(tpl_len(tpl_get($ctx, 'DataCorrections'))) ?></strong> perubahan data barang terbaru</p><span>Setiap baris memperlihatkan data yang diubah, nilai sebelum, dan nilai sesudah.</span></div>
  <div class="table-scroll-top" data-table-scroll-top aria-label="Geser tabel ke samping"><div></div></div>
  <div class="table-wrap" data-table-scroll-body>
    <table class="data-table reconciliation-change-table"><thead><tr><th>Tanggal</th><th>Inventory</th><th>Data yang berubah</th><th>Nilai sebelum</th><th>Nilai sesudah</th><th>Alasan dan petugas</th></tr></thead><tbody>
    <?php $__range1 = tpl_iter(tpl_get($ctx, 'DataCorrections')); if (count($__range1) > 0): $__parent1 = $ctx; foreach ($__range1 as $__key1 => $__item1): $ctx = $__item1; ?><?php $record = $ctx; ?><?php if (tpl_truthy(tpl_get($ctx, 'ChangeDetails'))): ?><?php $__range2 = tpl_iter(tpl_get($ctx, 'ChangeDetails')); if (count($__range2) > 0): $__parent2 = $ctx; foreach ($__range2 as $__key2 => $__item2): $ctx = $__item2; ?><tr>
      <td><strong><?= tpl_escape(tpl_datetime(tpl_get($record, 'CreatedAt'))) ?></strong></td>
      <td><strong><?= tpl_escape(tpl_get($record, 'InventoryReference')) ?></strong><small><?= tpl_escape(tpl_get($record, 'InventoryType')) ?></small></td>
      <td><span class="change-section-pill"><?= tpl_escape(tpl_change_section(tpl_get($ctx, 'Section'))) ?></span><strong><?= tpl_escape(tpl_change_field(tpl_get($ctx, 'Field'))) ?></strong><?php if (tpl_truthy(tpl_get($ctx, 'Context'))): ?><small><?= tpl_escape(tpl_get($ctx, 'Context')) ?></small><?php endif; ?></td>
      <td class="change-value before"><span><?= tpl_escape(tpl_change_value(tpl_get($ctx, 'Field'), tpl_get($ctx, 'Before'))) ?></span></td>
      <td class="change-value after"><span><?= tpl_escape(tpl_change_value(tpl_get($ctx, 'Field'), tpl_get($ctx, 'After'))) ?></span></td>
      <td><strong><?php if (tpl_truthy(tpl_get($record, 'CorrectionReason'))): ?><?= tpl_escape(tpl_get($record, 'CorrectionReason')) ?><?php else: ?>Alasan belum tersedia<?php endif; ?></strong><small><?= tpl_escape(tpl_get($record, 'Actor')) ?></small></td>
    </tr><?php $ctx = $__parent2; endforeach; endif; ?><?php else: ?><tr>
      <td><strong><?= tpl_escape(tpl_datetime(tpl_get($ctx, 'CreatedAt'))) ?></strong></td><td><strong><?= tpl_escape(tpl_get($ctx, 'InventoryReference')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'InventoryType')) ?></small></td>
      <td colspan="3"><strong>Rincian nilai sebelum dan sesudah belum tersedia</strong><small>Catatan ini dibuat sebelum migration audit perubahan data diterapkan.</small></td>
      <td><strong><?php if (tpl_truthy(tpl_get($ctx, 'CorrectionReason'))): ?><?= tpl_escape(tpl_get($ctx, 'CorrectionReason')) ?><?php else: ?>Alasan belum tersedia<?php endif; ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Actor')) ?></small></td>
    </tr><?php endif; ?><?php $ctx = $__parent1; endforeach; else: ?><tr><td colspan="6"><div class="empty-state small"><h3>Belum ada perubahan data barang</h3><p>Gunakan tombol Perubahan data barang untuk memperbarui data sekaligus menyimpan audit sebelum dan sesudah.</p></div></td></tr><?php endif; ?>
    </tbody></table>
  </div>
</section>
<?php else: ?>
<section class="panel table-panel">
  <div class="table-meta"><p><strong><?= tpl_escape(tpl_len(tpl_get($ctx, 'Reconciliations'))) ?></strong> hasil rekonsiliasi terbaru</p><span>Tab ini hanya berisi penyesuaian antara catatan aplikasi dan kondisi fisik di lapangan.</span></div>
  <div class="table-wrap">
    <table class="data-table"><thead><tr><th>Tanggal</th><th>Jenis rekonsiliasi</th><th>Inventory</th><th>Perubahan status</th><th>Catatan</th><th>Petugas</th></tr></thead><tbody>
    <?php $__range3 = tpl_iter(tpl_get($ctx, 'Reconciliations')); if (count($__range3) > 0): $__parent3 = $ctx; foreach ($__range3 as $__key3 => $__item3): $ctx = $__item3; ?><tr>
      <td><strong><?= tpl_escape(tpl_datetime(tpl_get($ctx, 'CreatedAt'))) ?></strong></td>
      <td><?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Type'), 'recorded_not_found'))): ?><span class="status-chip danger">Tercatat, tidak ada di lapangan</span><small>Dikeluarkan dari inventory aktif</small><?php else: ?><span class="status-chip success">Ada di lapangan, belum tercatat</span><small>Ditambahkan ke inventory</small><?php endif; ?></td>
      <td><strong><?= tpl_escape(tpl_get($ctx, 'InventoryReference')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'InventoryType')) ?></small></td>
      <td><strong><?php if (tpl_truthy(tpl_get($ctx, 'PreviousStatusLabel'))): ?><?= tpl_escape(tpl_get($ctx, 'PreviousStatusLabel')) ?><?php else: ?>-<?php endif; ?></strong><small>Menjadi: <?= tpl_escape(tpl_get($ctx, 'ResultStatusLabel')) ?></small></td>
      <td class="description-cell"><strong><?= tpl_escape(tpl_get($ctx, 'Notes')) ?></strong></td><td><?= tpl_escape(tpl_get($ctx, 'Actor')) ?></td>
    </tr><?php $ctx = $__parent3; endforeach; else: ?><tr><td colspan="6"><div class="empty-state small"><h3>Belum ada hasil rekonsiliasi</h3><p>Catat perbedaan antara inventory aplikasi dan kondisi fisik di lapangan.</p></div></td></tr><?php endif; ?>
    </tbody></table>
  </div>
</section>
<?php endif; ?>

<?php if (tpl_truthy(tpl_get($ctx, 'CanManage'))): ?>
<div class="modal action-modal" id="reconciliation-modal" role="dialog" aria-modal="true" aria-labelledby="reconciliation-modal-title" hidden>
  <div class="modal-backdrop" data-close-modal></div>
  <section class="modal-panel modal-panel-wide reconciliation-modal-panel correction-modal-panel">
    <header class="modal-header"><div><p class="eyebrow" data-reconciliation-modal-eyebrow>Rekonsiliasi inventory</p><h2 id="reconciliation-modal-title" data-reconciliation-modal-title>Sesuaikan catatan dengan kondisi sebenarnya</h2><p data-reconciliation-modal-description>Pilih jenis rekonsiliasi, cari barang yang terkait, lalu lengkapi data pemeriksaan.</p></div><button class="icon-button" type="button" data-close-modal aria-label="Tutup"><svg viewBox="0 0 24 24"><path d="m6 6 12 12M18 6 6 18"/></svg></button></header>
    <form method="post" action="/rekonsiliasi" enctype="multipart/form-data" data-reconciliation-form>
      <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($ctx, 'CSRF')) ?>">
      <div class="action-modal-scroll reconciliation-modal-scroll">
        <section class="action-field-section reconciliation-type-section" data-reconciliation-type-section><h3>Jenis rekonsiliasi</h3><div class="choice-question"><div>
          <label><input type="radio" name="reconciliation_type" value="recorded_not_found" required> Tercatat di aplikasi tetapi tidak ada di lapangan</label>
          <label><input type="radio" name="reconciliation_type" value="found_not_recorded" required> Tidak ada di aplikasi tetapi ditemukan di lapangan</label>
          <label data-reconciliation-correction-option hidden><input type="radio" name="reconciliation_type" value="data_correction" required> Perubahan data barang</label>
        </div></div></section>

        <section class="action-field-section reconciliation-inventory-picker" data-reconciliation-fields="recorded_not_found,data_correction" hidden>
          <div class="picker-heading"><div><h3 data-reconciliation-picker-title>Pilih inventory</h3><p data-reconciliation-picker-help>Cari berdasarkan nomor penetapan, nomor kontainer, jenis inventory, atau uraian barang.</p></div><strong data-reconciliation-picker-count>0 barang dipilih</strong></div>
          <div class="picker-controls reconciliation-picker-controls">
            <label class="search-field large"><svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/></svg><input type="search" data-reconciliation-search placeholder="Cari nomor penetapan, kontainer, jenis, atau uraian..." autocomplete="off"></label>
            <button type="button" class="picker-clear" data-reconciliation-search-clear disabled>Kosongkan pencarian</button>
          </div>
          <div class="multi-picker-list reconciliation-picker" data-reconciliation-list>
          <?php $__range4 = tpl_iter(tpl_get($ctx, 'EligibleItems')); if (count($__range4) > 0): $__parent4 = $ctx; foreach ($__range4 as $__key4 => $__item4): $ctx = $__item4; ?><label class="multi-picker-item" data-reconciliation-item data-active="<?= tpl_escape(tpl_get($ctx, 'IsActive')) ?>" data-search="<?= tpl_escape(tpl_lower(tpl_get($ctx, 'DeterminationNo'))) ?> <?= tpl_escape(tpl_lower(tpl_get($ctx, 'ReferenceNo'))) ?> <?= tpl_escape(tpl_lower(tpl_get($ctx, 'ContainerNo'))) ?> <?= tpl_escape(tpl_lower(tpl_get($ctx, 'Description'))) ?> <?= tpl_escape(tpl_lower(tpl_get($ctx, 'Type'))) ?> <?= tpl_escape(tpl_lower(tpl_get($ctx, 'StatusLabel'))) ?>"><input type="radio" name="inventory_id" value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>" disabled><span><strong><?= tpl_escape(tpl_get($ctx, 'DeterminationNo')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Type')) ?> · <?php if (tpl_truthy(tpl_get($ctx, 'ContainerNo'))): ?><?= tpl_escape(tpl_get($ctx, 'ContainerNo')) ?><?php else: ?>LCL<?php endif; ?></small><em><?= tpl_escape(tpl_get($ctx, 'Description')) ?></em></span><i><?= tpl_escape(tpl_get($ctx, 'StatusLabel')) ?></i></label><?php $ctx = $__parent4; endforeach; endif; ?>
          </div>
          <p class="picker-empty" data-reconciliation-empty hidden>Inventory tidak ditemukan. Ubah kata pencarian.</p>
        </section>

        <section class="action-field-section" data-reconciliation-fields="found_not_recorded" hidden>
          <div class="picker-heading"><div><h3>Tambah barang yang ditemukan di lapangan</h3><p>Isi identitas barang dan status yang menggambarkan kondisi sebenarnya saat pemeriksaan.</p></div></div>
          <div class="form-grid cols-3">
            <label>Nomor dokumen dasar <em>*</em><input name="determination_no" disabled></label>
            <label>Tanggal dokumen <em>*</em><input type="date" name="determination_date" value="<?= tpl_escape(tpl_isodate(tpl_get($ctx, 'Now'))) ?>" disabled></label>
            <label>Jenis inventory <em>*</em><select name="item_type" data-reconciliation-item-type disabled><option value="">Pilih jenis</option><?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'inventory.type.btd'))): ?><option value="BTD">BTD</option><?php endif; ?><?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'inventory.type.bdn'))): ?><option value="BDN">BDN</option><?php endif; ?><?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'inventory.type.bmmn'))): ?><option value="BMMN">BMMN</option><?php endif; ?><?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'inventory.type.titipan'))): ?><option value="TITIPAN">Barang Titipan</option><?php endif; ?></select></label>
            <label data-reconciliation-bdn hidden>Kategori BDN<select name="category" disabled><option value="">Pilih kategori</option><?php $__range5 = tpl_iter(tpl_get($ctx, 'BDNCategoryNames')); if (count($__range5) > 0): $__parent5 = $ctx; foreach ($__range5 as $__key5 => $__item5): $ctx = $__item5; ?><option><?= tpl_escape($ctx) ?></option><?php $ctx = $__parent5; endforeach; endif; ?></select></label>
            <label data-reconciliation-titipan hidden>Kategori barang titipan<select name="entrusted_category" disabled><option value="">Pilih kategori</option><?php $__range6 = tpl_iter(tpl_get($ctx, 'EntrustedCategoryNames')); if (count($__range6) > 0): $__parent6 = $ctx; foreach ($__range6 as $__key6 => $__item6): $ctx = $__item6; ?><option><?= tpl_escape($ctx) ?></option><?php $ctx = $__parent6; endforeach; endif; ?></select></label>
            <label data-reconciliation-titipan hidden>Kantor/unit asal<input name="source_office" disabled placeholder="Nama kantor atau unit penitip"></label>
            <label data-reconciliation-tps>TPS asal<select name="origin_warehouse" disabled><option value="">Pilih TPS</option><?php $__range7 = tpl_iter(tpl_get($ctx, 'TPSNames')); if (count($__range7) > 0): $__parent7 = $ctx; foreach ($__range7 as $__key7 => $__item7): $ctx = $__item7; ?><option><?= tpl_escape($ctx) ?></option><?php $ctx = $__parent7; endforeach; endif; ?></select></label>
            <label>TPP lokasi fisik <em>*</em><select name="facility_id" disabled><option value="">Pilih TPP</option><?php $__range8 = tpl_iter(tpl_get($ctx, 'Facilities')); if (count($__range8) > 0): $__parent8 = $ctx; foreach ($__range8 as $__key8 => $__item8): $ctx = $__item8; ?><option value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>"><?= tpl_escape(tpl_get($ctx, 'Name')) ?></option><?php $ctx = $__parent8; endforeach; endif; ?></select></label>
            <label>Blok/gudang<input name="location" disabled></label>
            <label>Nomor manifest<input name="manifest_no" disabled></label><label>Tanggal manifest<input type="date" name="manifest_date" disabled></label><label>Pos manifest<input name="manifest_position" disabled></label>
            <label>Jenis muatan <em>*</em><select name="load_type" data-reconciliation-load disabled><option value="">Pilih</option><option value="FCL">FCL</option><option value="LCL">LCL</option></select></label>
            <label data-reconciliation-fcl hidden>Nomor kontainer<input name="container_no" disabled></label><label data-reconciliation-fcl hidden>Ukuran<select name="container_size" disabled><option value="">Pilih ukuran</option><?php $__range9 = tpl_iter(tpl_get($ctx, 'ContainerSizeOptions')); if (count($__range9) > 0): $__parent9 = $ctx; foreach ($__range9 as $__key9 => $__item9): $ctx = $__item9; ?><option value="<?= tpl_escape(tpl_get($ctx, 'Code')) ?>"><?= tpl_escape(tpl_get($ctx, 'Label')) ?></option><?php $ctx = $__parent9; endforeach; endif; ?></select></label>
            <label data-reconciliation-lcl hidden>Perkiraan volume m³<input type="number" step="0.01" min="0.01" name="estimated_volume_m3" disabled></label>
            <label class="span-3">Uraian barang <em>*</em><textarea name="description" rows="2" disabled></textarea></label>
            <label>Jenis barang <em>*</em><select name="item_kind" disabled><option value="">Pilih jenis</option><?php $__range10 = tpl_iter(tpl_get($ctx, 'ItemKindNames')); if (count($__range10) > 0): $__parent10 = $ctx; foreach ($__range10 as $__key10 => $__item10): $ctx = $__item10; ?><option><?= tpl_escape($ctx) ?></option><?php $ctx = $__parent10; endforeach; endif; ?></select></label>
            <label>Jumlah <em>*</em><input type="number" step="0.01" min="0.01" name="quantity" disabled></label>
            <label>Satuan <em>*</em><select name="unit" disabled><option value="">Pilih satuan</option><?php $__range11 = tpl_iter(tpl_get($ctx, 'UnitNames')); if (count($__range11) > 0): $__parent11 = $ctx; foreach ($__range11 as $__key11 => $__item11): $ctx = $__item11; ?><option><?= tpl_escape($ctx) ?></option><?php $ctx = $__parent11; endforeach; endif; ?></select></label>
            <label>Nilai barang<input inputmode="numeric" name="goods_value" disabled></label>
            <label>Status sebenarnya <em>*</em><select name="initial_status_code" disabled><option value="">Pilih status</option><optgroup label="Inventory"><option value="ditetapkan">Baru ditetapkan</option><option value="pencacahan">Selesai pencacahan</option><option value="request_penelitian_pfpd">Request Penelitian PFPD</option><option value="penelitian_pfpd">Penelitian PFPD</option><option value="bmmn_aktif">BMMN aktif</option><option value="barang_titipan_aktif">Barang titipan aktif</option></optgroup><optgroup label="Lelang"><option value="kep_lelang">KEP Lelang</option><option value="kep_htl">KEP Harga Terendah Lelang</option><option value="jadwal_lelang">Jadwal lelang</option><option value="laku">Laku</option><option value="tidak_laku">Tidak laku</option><option value="alokasi_hasil_lelang">Alokasi hasil lelang</option></optgroup><optgroup label="Musnah"><option value="kep_musnah">KEP Musnah</option><option value="ba_musnah">BA Musnah</option></optgroup><optgroup label="Hibah/PSP"><option value="ba_serah_terima_hibah">BA Serah Terima Hibah</option><option value="ba_serah_terima_psp">BA Serah Terima PSP</option></optgroup></select></label>
          </div>
        </section>

        <section class="action-field-section correction-editor-section" data-reconciliation-fields="data_correction" hidden>
          <input type="hidden" name="correction_item_json" data-correction-item-json disabled>
          <input type="hidden" name="correction_events_json" data-correction-events-json disabled>
          <input type="hidden" name="correction_processes_json" data-correction-processes-json disabled>
          <div class="correction-loading" data-correction-loading hidden><span></span><strong>Memuat data barang dan dokumen...</strong></div>
          <div data-correction-empty><div class="empty-state small"><h3>Pilih satu barang</h3><p>Form perubahan akan terbuka setelah barang dipilih.</p></div></div>
          <div class="correction-editor" data-correction-editor hidden>
            <div class="correction-warning"><strong>Perubahan langsung berlaku</strong><p>ID sistem, status alur, dan status aktif tidak dapat diedit agar konsistensi proses tetap terjaga. Seluruh data bisnis, nomor dokumen, tanggal, nilai, dan catatan dapat diperbarui.</p></div>

            <section class="correction-group"><header><span>01</span><div><h3>Identitas dan dokumen dasar</h3><p>Perbarui nomor referensi, BCF/penetapan, manifest, dan klasifikasi inventory.</p></div></header><div class="form-grid cols-3">
              <label>Nomor referensi <em>*</em><input data-correction-field="reference_no" data-required="true"></label>
              <label>Jenis inventory <em>*</em><select data-correction-field="item_type" data-required="true"><option value="BTD">BTD</option><option value="BDN">BDN</option><option value="BMMN">BMMN</option><option value="TITIPAN">Barang Titipan</option></select></label>
              <label>Jenis asal <em>*</em><select data-correction-field="origin_type" data-required="true"><option value="BTD">BTD</option><option value="BDN">BDN</option><option value="BMMN">BMMN</option><option value="TITIPAN">Barang Titipan</option></select></label>
              <label>Nomor penetapan/BCF <em>*</em><input data-correction-field="determination_no" data-required="true"></label>
              <label>Tanggal penetapan/BCF <em>*</em><input type="date" data-correction-field="determination_date" data-correction-date data-required="true"></label>
              <label>Nomor BL<input data-correction-field="bl_no"></label>
              <label>Tanggal BL<input type="date" data-correction-field="bl_date" data-correction-date></label>
              <label>Kategori<input data-correction-field="category"></label>
              <label>Nomor manifest<input data-correction-field="manifest_no"></label>
              <label>Tanggal manifest<input type="date" data-correction-field="manifest_date" data-correction-date></label>
              <label>Pos manifest<input data-correction-field="manifest_position"></label>
              <label>Kategori barang titipan<input data-correction-field="entrusted_category"></label>
              <label>Kantor/unit asal<input data-correction-field="source_office"></label>
            </div></section>

            <section class="correction-group"><header><span>02</span><div><h3>Identitas barang dan pemilik</h3><p>Seluruh uraian, kuantitas, nilai, kondisi, dan identitas pemilik dapat dikoreksi.</p></div></header><div class="form-grid cols-3">
              <label class="span-3">Uraian barang <em>*</em><textarea rows="3" data-correction-field="description" data-required="true"></textarea></label>
              <label>Jenis barang <em>*</em><select data-correction-field="item_kind" data-required="true"><?php $__range12 = tpl_iter(tpl_get($ctx, 'ItemKindNames')); if (count($__range12) > 0): $__parent12 = $ctx; foreach ($__range12 as $__key12 => $__item12): $ctx = $__item12; ?><option><?= tpl_escape($ctx) ?></option><?php $ctx = $__parent12; endforeach; endif; ?></select></label>
              <label>Kondisi barang<select data-correction-field="goods_condition"><option value="">Belum ditetapkan</option><?php $__range13 = tpl_iter(tpl_get($ctx, 'GoodsConditionNames')); if (count($__range13) > 0): $__parent13 = $ctx; foreach ($__range13 as $__key13 => $__item13): $ctx = $__item13; ?><option><?= tpl_escape($ctx) ?></option><?php $ctx = $__parent13; endforeach; endif; ?></select></label>
              <label>Nilai barang<input inputmode="numeric" data-correction-field="goods_value" data-correction-money></label>
              <label>Jumlah<input type="number" min="0" step="0.01" data-correction-field="quantity" data-correction-number></label>
              <label>Satuan<select data-correction-field="unit"><option value="">Pilih satuan</option><?php $__range14 = tpl_iter(tpl_get($ctx, 'UnitNames')); if (count($__range14) > 0): $__parent14 = $ctx; foreach ($__range14 as $__key14 => $__item14): $ctx = $__item14; ?><option><?= tpl_escape($ctx) ?></option><?php $ctx = $__parent14; endforeach; endif; ?></select></label>
              <label>Nama pemilik<input data-correction-field="owner_name"></label>
              <label class="span-2">Alamat pemilik<textarea rows="2" data-correction-field="owner_address"></textarea></label>
            </div></section>

            <section class="correction-group"><header><span>03</span><div><h3>Lokasi dan unit fisik</h3><p>Koreksi TPS/TPP, blok, kontainer, ukuran, dan volume tanpa mengubah status alur.</p></div></header><div class="form-grid cols-3">
              <label>TPS/gudang asal<input data-correction-field="origin_warehouse"></label>
              <label>TPP<select data-correction-field="facility_id"><option value="">Belum ditentukan</option><?php $__range15 = tpl_iter(tpl_get($ctx, 'Facilities')); if (count($__range15) > 0): $__parent15 = $ctx; foreach ($__range15 as $__key15 => $__item15): $ctx = $__item15; ?><option value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>"><?= tpl_escape(tpl_get($ctx, 'Name')) ?></option><?php $ctx = $__parent15; endforeach; endif; ?></select></label>
              <label>Blok/lokasi<input data-correction-field="location"></label>
              <label>Status lokasi<input data-correction-field="location_status"></label>
              <label>Jenis muatan <em>*</em><select data-correction-field="load_type" data-required="true"><option value="FCL">FCL</option><option value="LCL">LCL</option></select></label>
              <label>Nomor kontainer<input data-correction-field="container_no"></label>
              <label>Ukuran kontainer<select data-correction-field="container_size"><option value="">Tidak ada</option><?php $__range16 = tpl_iter(tpl_get($ctx, 'ContainerSizeOptions')); if (count($__range16) > 0): $__parent16 = $ctx; foreach ($__range16 as $__key16 => $__item16): $ctx = $__item16; ?><option value="<?= tpl_escape(tpl_get($ctx, 'Code')) ?>"><?= tpl_escape(tpl_get($ctx, 'Label')) ?></option><?php $ctx = $__parent16; endforeach; endif; ?></select></label>
              <label>Perkiraan volume m³<input type="number" min="0" step="0.01" data-correction-field="estimated_volume_m3" data-correction-number></label>
              <label>ID unit fisik<input data-correction-field="physical_unit_id"></label>
              <label class="checkbox-card"><input type="checkbox" data-correction-field="at_tpp" data-correction-checkbox><span>Barang berada di TPP</span></label>
              <label class="checkbox-card"><input type="checkbox" data-correction-field="occupancy_primary" data-correction-checkbox><span>Unit utama perhitungan kapasitas</span></label>
            </div></section>

            <section class="correction-group"><header><span>04</span><div><h3>Penelitian, dokumen asal, dan peruntukan</h3><p>Nomor request, HS, lartas, usulan, persetujuan, serta dokumen pengeluaran dapat diubah.</p></div></header><div class="form-grid cols-3">
              <label class="checkbox-card"><input type="checkbox" data-correction-field="pfpd_required" data-correction-checkbox><span>Memerlukan penelitian PFPD</span></label>
              <label>Nomor request PFPD<input data-correction-field="research_request_no"></label>
              <label>Tanggal request PFPD<input type="date" data-correction-field="research_request_date" data-correction-date></label>
              <label>HS Code<input data-correction-field="hs_code"></label>
              <label class="checkbox-card"><input type="checkbox" data-correction-field="is_restricted" data-correction-checkbox><span>Terkena ketentuan lartas</span></label>
              <label>Keterangan lartas<input data-correction-field="restriction_rule"></label>
              <label>Jenis dokumen asal<input data-correction-field="origin_document_type"></label>
              <label>Nomor dokumen asal<input data-correction-field="origin_document_no"></label>
              <label>Tanggal dokumen asal<input type="date" data-correction-field="origin_document_date" data-correction-date></label>
              <label>Peruntukan<select data-correction-field="allocation_purpose"><option value="">Belum ditentukan</option><?php $__range17 = tpl_iter(tpl_get($ctx, 'AllocationPurposeNames')); if (count($__range17) > 0): $__parent17 = $ctx; foreach ($__range17 as $__key17 => $__item17): $ctx = $__item17; ?><option><?= tpl_escape($ctx) ?></option><?php $ctx = $__parent17; endforeach; endif; ?></select></label>
              <label>Jenis dokumen usulan<input data-correction-field="allocation_proposal_type"></label>
              <label>Nomor dokumen usulan<input data-correction-field="allocation_proposal_no"></label>
              <label>Tanggal dokumen usulan<input type="date" data-correction-field="allocation_proposal_date" data-correction-date></label>
              <label>Jenis dokumen persetujuan<input data-correction-field="allocation_approval_type"></label>
              <label>Nomor dokumen persetujuan<input data-correction-field="allocation_approval_no"></label>
              <label>Tanggal persetujuan<input type="date" data-correction-field="allocation_approval_date" data-correction-date></label>
              <label>Nomor dokumen pengeluaran<input data-correction-field="exit_document_no"></label>
              <label>Tanggal pengeluaran<input type="date" data-correction-field="exit_document_date" data-correction-date></label>
              <label>Jenis pengeluaran<input data-correction-field="exit_type"></label>
              <label class="span-3">Catatan pengeluaran<textarea rows="2" data-correction-field="exit_notes"></textarea></label>
            </div></section>

            <section class="correction-group"><header><span>05</span><div><h3>Nomor surat dan timeline</h3><p>Edit nomor dan tanggal surat, ND, KEP, BA, risalah, label, serta catatan pada setiap tahapan.</p></div></header><div class="correction-record-list" data-correction-event-list></div></section>
            <section class="correction-group"><header><span>06</span><div><h3>Data proses</h3><p>Edit nilai HTL, hasil lelang, ND jadwal, penerima, biaya musnah, dan data serah terima.</p></div></header><div class="correction-record-list" data-correction-process-list></div></section>

            <section class="correction-reason-section">
              <h3>Alasan perubahan <em>*</em></h3>
              <label class="correction-reason-field">
                <span>Dasar koreksi</span>
                <select name="correction_reason" data-correction-reason disabled>
                  <option value="">Pilih alasan perubahan</option>
                  <option value="Kesalahan input">Kesalahan input</option>
                  <option value="Error pada saat pengisian awal">Error pada saat pengisian awal</option>
                </select>
              </label>
              <label class="document-upload-field">Upload dokumen pendukung <span class="optional-mark">Opsional</span><input type="file" name="document_file" accept="application/pdf,image/jpeg,image/png,image/webp,image/gif" data-correction-upload disabled><small>Dokumen akan dicatat pada timeline perubahan data barang.</small></label>
            </section>
          </div>
        </section>

        <section class="action-field-section reconciliation-notes-section" data-reconciliation-notes><h3>Catatan pemeriksaan</h3><label>Catatan rekonsiliasi <em>*</em><textarea name="notes" rows="4" required placeholder="Jelaskan hasil pemeriksaan fisik dan dasar koreksi inventory"></textarea></label><label class="document-upload-field">Upload dokumen pendukung <span class="optional-mark">Opsional</span><input type="file" name="document_file" accept="application/pdf,image/jpeg,image/png,image/webp,image/gif"><small>PDF atau gambar, maksimal 8 MB. File dapat diunduh melalui timeline barang.</small></label></section>
      </div>
      <footer class="modal-footer reconciliation-modal-footer"><button class="button ghost" type="button" data-close-modal>Batal</button><button class="button primary" type="submit" data-reconciliation-submit>Simpan rekonsiliasi</button></footer>
    </form>
  </section>
</div>
<?php endif; ?>
