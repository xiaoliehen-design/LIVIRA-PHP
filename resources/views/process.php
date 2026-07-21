<section class="page-intro page-intro-actions">
  <div><p><?= tpl_escape(tpl_get($ctx, 'Subtitle')) ?></p></div>
  <div class="intro-actions">
    <?php if (tpl_truthy(tpl_get($ctx, 'History'))): ?>
    <a class="button secondary" href="/proses/<?= tpl_escape(tpl_get($ctx, 'ProcessType')) ?>"><svg viewBox="0 0 24 24"><path d="m15 18-6-6 6-6"/></svg>Kembali ke proses aktif</a>
    <?php else: ?>
    <a class="button secondary history-button" href="/proses/<?= tpl_escape(tpl_get($ctx, 'ProcessType')) ?>?history=1"><svg viewBox="0 0 24 24"><path d="M3 12a9 9 0 1 0 3-6.7M3 4v5h5"/><path d="M12 7v5l3 2"/></svg>History</a>
    <?php if (tpl_truthy(tpl_get($ctx, 'CanManage'))): ?><button class="button primary" type="button" data-open-process-action><svg viewBox="0 0 24 24"><path d="M5 7h14M5 12h14M5 17h14"/></svg>Action</button><?php endif; ?>
    <?php endif; ?>
  </div>
</section>

<section class="panel table-panel">
  <form class="table-toolbar" method="get" action="/proses/<?= tpl_escape(tpl_get($ctx, 'ProcessType')) ?>">
    <?php if (tpl_truthy(tpl_get($ctx, 'History'))): ?><input type="hidden" name="history" value="1"><?php endif; ?>
    <input type="hidden" name="page_size" value="<?= tpl_escape(tpl_get($ctx, 'Pagination.PageSize')) ?>">
    <label class="search-field"><svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/></svg><input type="search" name="q" value="<?= tpl_escape(tpl_get($ctx, 'Query')) ?>" placeholder="Cari kontainer, nomor penetapan, atau uraian barang…"></label>
    <label class="select-field"><span>TPP</span><select name="tpp" data-auto-submit><option value="">Semua TPP</option><?php $__range1 = tpl_iter(tpl_get($ctx, 'Facilities')); if (count($__range1) > 0): $__parent1 = $ctx; foreach ($__range1 as $__key1 => $__item1): $ctx = $__item1; ?><option value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>" <?php if (tpl_truthy(tpl_eq(tpl_get($root, 'FacilityID'), tpl_get($ctx, 'ID')))): ?>selected<?php endif; ?>><?= tpl_escape(tpl_get($ctx, 'Name')) ?></option><?php $ctx = $__parent1; endforeach; endif; ?></select></label>
    <?php if (tpl_truthy(tpl_get($ctx, 'History'))): ?><label class="select-field"><span>Arsip</span><select disabled><option>Seluruh riwayat</option></select></label><?php else: ?><label class="select-field"><span>Status</span><select name="status" data-auto-submit><option value="">Semua proses</option><option value="active" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Status'), 'active'))): ?>selected<?php endif; ?>>Masih berjalan</option><option value="completed" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Status'), 'completed'))): ?>selected<?php endif; ?>>Selesai</option></select></label><?php endif; ?>
    <label class="select-field sort-field"><span>Urutkan</span><select name="sort" data-auto-submit><option value="newest" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Sort'), 'newest'))): ?>selected<?php endif; ?>>Pembaruan terbaru</option><option value="determination_newest" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Sort'), 'determination_newest'))): ?>selected<?php endif; ?>>Penetapan terbaru</option><option value="determination_oldest" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Sort'), 'determination_oldest'))): ?>selected<?php endif; ?>>Penetapan terlama</option><option value="value_desc" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Sort'), 'value_desc'))): ?>selected<?php endif; ?>><?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ProcessType'), 'lelang'))): ?>HTL tertinggi<?php else: ?>Nilai tertinggi<?php endif; ?></option><option value="value_asc" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Sort'), 'value_asc'))): ?>selected<?php endif; ?>><?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ProcessType'), 'lelang'))): ?>HTL terendah<?php else: ?>Nilai terendah<?php endif; ?></option><option value="oldest" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Sort'), 'oldest'))): ?>selected<?php endif; ?>>Pembaruan terlama</option></select></label>
    <button class="button compact filter-button" type="submit">Terapkan</button>
  </form>
  <div class="table-meta"><p>Menampilkan <strong><?= tpl_escape(tpl_get($ctx, 'Pagination.StartItem')) ?>–<?= tpl_escape(tpl_get($ctx, 'Pagination.EndItem')) ?></strong> dari <?= tpl_escape(tpl_get($ctx, 'Pagination.TotalItems')) ?> <?php if (tpl_truthy(tpl_get($ctx, 'History'))): ?>riwayat<?php else: ?>proses<?php endif; ?> <?= tpl_escape(tpl_get($ctx, 'ProcessSingular')) ?></p><span><i class="legend-dot <?= tpl_escape(tpl_get($ctx, 'ProcessType')) ?>"></i>Status kanan dapat diklik untuk melihat timestamp</span></div>
  <div class="table-scroll-top" data-table-scroll-top aria-label="Geser tabel ke samping"><div></div></div>
  <div class="table-pagination">
    <div class="page-size-picker"><span>Tampilkan</span><?php $__range2 = tpl_iter(tpl_get($ctx, 'Pagination.Sizes')); if (count($__range2) > 0): $__parent2 = $ctx; foreach ($__range2 as $__key2 => $__item2): $ctx = $__item2; ?><a class="<?php if (tpl_truthy(tpl_get($ctx, 'Selected'))): ?>active<?php endif; ?>" href="<?= tpl_escape(tpl_get($ctx, 'URL')) ?>"><?= tpl_escape(tpl_get($ctx, 'Value')) ?></a><?php $ctx = $__parent2; endforeach; endif; ?><span>baris</span></div>
    <div class="page-navigation"><span>Halaman <?= tpl_escape(tpl_get($ctx, 'Pagination.Page')) ?> dari <?= tpl_escape(tpl_get($ctx, 'Pagination.TotalPages')) ?> · Total <?= tpl_escape(tpl_get($ctx, 'Pagination.TotalItems')) ?> item</span><?php if (tpl_truthy(tpl_get($ctx, 'Pagination.HasPrevious'))): ?><a href="<?= tpl_escape(tpl_get($ctx, 'Pagination.PreviousURL')) ?>">Sebelumnya</a><?php else: ?><span class="disabled">Sebelumnya</span><?php endif; ?><?php if (tpl_truthy(tpl_get($ctx, 'Pagination.HasNext'))): ?><a href="<?= tpl_escape(tpl_get($ctx, 'Pagination.NextURL')) ?>">Berikutnya</a><?php else: ?><span class="disabled">Berikutnya</span><?php endif; ?></div>
  </div>
  <div class="table-wrap" data-table-scroll-body>
    <table class="data-table process-table">
      <thead><tr><th>Kontainer & inventory</th><th>Uraian barang</th><th>Lokasi</th><?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ProcessType'), 'lelang'))): ?><th>HTL / hasil lelang</th><?php elseif (tpl_truthy(tpl_eq(tpl_get($ctx, 'ProcessType'), 'musnah'))): ?><th>Biaya musnah</th><?php else: ?><th>Jenis serah terima</th><?php endif; ?><th class="status-column">Status proses</th></tr></thead>
      <tbody>
      <?php $__range3 = tpl_iter(tpl_get($ctx, 'Processes')); if (count($__range3) > 0): $__parent3 = $ctx; foreach ($__range3 as $__key3 => $__item3): $ctx = $__item3; ?>
        <tr>
          <td><strong><?php if (tpl_truthy(tpl_get($ctx, 'Inventory.ContainerNo'))): ?><?= tpl_escape(tpl_get($ctx, 'Inventory.ContainerNo')) ?> · <?= tpl_escape(tpl_get($ctx, 'Inventory.ContainerSize')) ?>'<?php else: ?>LCL · <?= tpl_escape(tpl_number(tpl_get($ctx, 'Inventory.EstimatedVolumeM3'))) ?> m³<?php endif; ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Inventory.DeterminationNo')) ?> · <?= tpl_escape(tpl_get($ctx, 'Inventory.Type')) ?></small></td>
          <td class="description-cell"><strong><?= tpl_escape(tpl_get($ctx, 'Inventory.Description')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Inventory.OwnerName')) ?></small></td>
          <td><span class="facility-name"><?= tpl_escape(tpl_get($ctx, 'Inventory.LocationStatus')) ?></span><small><?= tpl_escape(tpl_get($ctx, 'Inventory.Location')) ?></small></td>
          <?php if (tpl_truthy(tpl_eq(tpl_get($root, 'ProcessType'), 'lelang'))): ?><td><strong><?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'HTLValue'))) ?></strong><small>Harga Terendah Lelang</small><small>Hasil lelang: <?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'SaleValue'))) ?></small><?php if (tpl_truthy(tpl_get($ctx, 'ScheduleDocumentNo'))): ?><small>ND Jadwal: <?= tpl_escape(tpl_get($ctx, 'ScheduleDocumentNo')) ?></small><?php endif; ?><?php if (tpl_truthy(tpl_get($ctx, 'AllocationTarget'))): ?><small>Alokasi: <?= tpl_escape(tpl_get($ctx, 'AllocationTarget')) ?></small><?php else: ?><small>Pelaksanaan: <?= tpl_escape(tpl_date(tpl_get($ctx, 'ExecutionStartDate'))) ?> – <?= tpl_escape(tpl_date(tpl_get($ctx, 'ExecutionEndDate'))) ?></small><?php endif; ?></td><?php elseif (tpl_truthy(tpl_eq(tpl_get($root, 'ProcessType'), 'musnah'))): ?><td><strong><?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'DestructionCost'))) ?></strong><small>Biaya tercatat</small></td><?php else: ?><td><strong><?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'TransferType'), 'hibah'))): ?>HIBAH<?php elseif (tpl_truthy(tpl_eq(tpl_get($ctx, 'TransferType'), 'psp'))): ?>PSP<?php else: ?>Belum ditetapkan<?php endif; ?></strong><small>Hibah / PSP</small></td><?php endif; ?>
          <td class="status-column"><button class="status-button <?= tpl_escape(tpl_status_tone(tpl_get($ctx, 'StatusCode'))) ?>" type="button" data-timeline-url="/api/proses/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/timeline"><span><?= tpl_escape(tpl_get($ctx, 'StatusLabel')) ?></span><svg viewBox="0 0 24 24"><path d="m9 18 6-6-6-6"/></svg></button><small>Diperbarui <?= tpl_escape(tpl_date(tpl_get($ctx, 'UpdatedAt'))) ?></small></td>
        </tr>
      <?php $ctx = $__parent3; endforeach; else: ?><tr><td colspan="5"><div class="empty-state"><span><svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/></svg></span><?php if (tpl_truthy(tpl_get($root, 'History'))): ?><h3>Belum ada riwayat <?= tpl_escape(tpl_get($root, 'ProcessSingular')) ?></h3><p>Data akan berpindah ke halaman ini setelah proses mencapai tahap penyelesaian.</p><?php else: ?><h3>Belum ada proses <?= tpl_escape(tpl_get($root, 'ProcessSingular')) ?></h3><p>Klik Action dan pilih barang dari inventory.</p><?php endif; ?></div></td></tr><?php endif; ?>
      </tbody>
    </table>
  </div>
</section>

<?php if (tpl_truthy(tpl_not(tpl_get($ctx, 'History')))): ?>
<?php if (tpl_truthy(tpl_get($ctx, 'CanManage'))): ?>
<div class="modal action-modal" id="process-action-drawer" role="dialog" aria-modal="true" aria-labelledby="process-action-title" hidden>
  <div class="modal-backdrop" data-close-modal></div>
  <section class="modal-panel modal-panel-wide action-modal-panel">
    <header class="modal-header"><div><p class="eyebrow">Action <?= tpl_escape(tpl_get($ctx, 'ProcessSingular')) ?></p><h2 id="process-action-title" data-action-modal-title data-default-title="Pilih submenu action <?= tpl_escape(tpl_get($ctx, 'ProcessSingular')) ?>">Pilih submenu action <?= tpl_escape(tpl_get($ctx, 'ProcessSingular')) ?></h2><p data-action-modal-description data-default-description="Pilih action yang ingin dikerjakan. Form lengkap akan terbuka setelah submenu dipilih.">Pilih action yang ingin dikerjakan. Form lengkap akan terbuka setelah submenu dipilih.</p></div><button class="icon-button" type="button" data-close-modal aria-label="Tutup"><svg viewBox="0 0 24 24"><path d="m6 6 12 12M18 6 6 18"/></svg></button></header>
    <form method="post" action="/proses/<?= tpl_escape(tpl_get($ctx, 'ProcessType')) ?>/bulk-action" enctype="multipart/form-data" data-process-bulk-form data-process-type="<?= tpl_escape(tpl_get($ctx, 'ProcessType')) ?>">
      <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($ctx, 'CSRF')) ?>"><input type="hidden" name="event_code" data-event-code>
      <div class="action-modal-scroll">
        <section class="action-menu-stage" data-action-menu>
          <div class="action-menu-intro"><strong>Pilih jenis action</strong><p>Klik salah satu submenu di bawah. Setelah dipilih, sistem menampilkan form action dalam popup yang lebih lebar dan mudah dikerjakan.</p></div>
          <div class="step-picker action-step-grid">
            <?php $__range4 = tpl_iter(tpl_get($ctx, 'ProcessActions')); if (count($__range4) > 0): $__parent4 = $ctx; foreach ($__range4 as $__key4 => $__item4): $ctx = $__item4; ?><button type="button" class="step-option <?php if (tpl_truthy(tpl_get($ctx, 'CreatesProcess'))): ?>highlight<?php endif; ?>" data-step-code="<?= tpl_escape(tpl_get($ctx, 'Code')) ?>" data-step-label="<?= tpl_escape(tpl_get($ctx, 'Label')) ?>" data-step-description="<?= tpl_escape(tpl_get($ctx, 'Description')) ?>" data-step-document="<?= tpl_escape(tpl_get($ctx, 'Document')) ?>" data-creates-process="<?= tpl_escape(tpl_get($ctx, 'CreatesProcess')) ?>" data-allowed-status="<?= tpl_escape(tpl_get($ctx, 'AllowedStatus')) ?>"><span class="step-check"><svg viewBox="0 0 24 24"><path d="m5 12 4 4L19 6"/></svg></span><span><strong><?= tpl_escape(tpl_get($ctx, 'Label')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Description')) ?></small></span><svg class="step-arrow" viewBox="0 0 24 24"><path d="m9 18 6-6-6-6"/></svg></button><?php $ctx = $__parent4; endforeach; endif; ?>
          </div>
        </section>
        <section class="action-detail-stage" data-action-detail hidden>
          <div class="action-detail-toolbar"><button class="button ghost compact action-back-button" type="button" data-back-action-menu><svg viewBox="0 0 24 24"><path d="m15 18-6-6 6-6"/></svg>Kembali ke daftar action</button><div class="selected-step" data-selected-step hidden><span>Action dipilih</span><strong data-selected-step-label></strong></div></div>

        <section class="action-field-section inventory-multi-picker" data-process-picker hidden>
          <div class="picker-heading"><div><h3>Pilih barang</h3><p data-process-picker-help><?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ProcessType'), 'musnah'))): ?>Pilih barang aktif atau barang lelang berstatus Tidak Laku. Barang Tidak Laku yang dipilih akan dipindahkan dari lelang ke pemusnahan.<?php elseif (tpl_truthy(tpl_eq(tpl_get($ctx, 'ProcessType'), 'hibah'))): ?>Pilih barang aktif atau barang lelang berstatus Tidak Laku. Barang Tidak Laku yang dipilih akan dipindahkan dari lelang ke hibah/PSP.<?php else: ?>Pilih kontainer yang akan diproses.<?php endif; ?></p></div><strong data-process-picker-count>0 barang dipilih</strong></div>
          <div class="picker-controls">
            <label class="search-field large"><svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/></svg><input type="search" data-process-picker-search placeholder="Cari kontainer, penetapan, atau uraian…" autocomplete="off"></label>
            <div class="picker-selection-toolbar"><label class="picker-select-all"><input type="checkbox" data-process-picker-select-all><span>Pilih semua hasil yang tampil</span></label><button type="button" class="picker-clear" data-process-picker-clear>Kosongkan pilihan</button></div>
          </div>
          <div class="multi-picker-list" data-process-picker-list>
            <?php $__range5 = tpl_iter(tpl_get($ctx, 'EligibleItems')); if (count($__range5) > 0): $__parent5 = $ctx; foreach ($__range5 as $__key5 => $__item5): $ctx = $__item5; ?>
            <label class="multi-picker-item" data-process-candidate data-candidate-source="inventory" data-status="" data-search="<?= tpl_escape(tpl_lower(tpl_get($ctx, 'DeterminationNo'))) ?> <?= tpl_escape(tpl_lower(tpl_get($ctx, 'ContainerNo'))) ?> <?= tpl_escape(tpl_lower(tpl_get($ctx, 'Description'))) ?>" hidden>
              <input type="checkbox" name="inventory_ids[]" value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>" data-process-candidate-checkbox disabled>
              <span><strong><?php if (tpl_truthy(tpl_get($ctx, 'ContainerNo'))): ?><?= tpl_escape(tpl_get($ctx, 'ContainerNo')) ?> · <?= tpl_escape(tpl_get($ctx, 'ContainerSize')) ?>'<?php else: ?>LCL · <?= tpl_escape(tpl_number(tpl_get($ctx, 'EstimatedVolumeM3'))) ?> m³<?php endif; ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Type')) ?> · <?= tpl_escape(tpl_get($ctx, 'DeterminationNo')) ?></small><em><?= tpl_escape(tpl_get($ctx, 'Description')) ?></em></span><i><?php if (tpl_truthy(tpl_and(tpl_eq(tpl_get($ctx, 'CurrentDisposition'), 'lelang'), tpl_eq(tpl_get($ctx, 'StatusCode'), 'tidak_laku')))): ?>Tidak Laku · dialihkan<?php else: ?><?= tpl_escape(tpl_get($ctx, 'StatusLabel')) ?><?php endif; ?></i>
            </label>
            <?php $ctx = $__parent5; endforeach; endif; ?>
            <?php $__range6 = tpl_iter(tpl_get($ctx, 'CandidateProcesses')); if (count($__range6) > 0): $__parent6 = $ctx; foreach ($__range6 as $__key6 => $__item6): $ctx = $__item6; ?>
            <label class="multi-picker-item" data-process-candidate data-candidate-source="process" data-process-id="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>" data-status="<?= tpl_escape(tpl_get($ctx, 'StatusCode')) ?>" data-active="<?= tpl_escape(tpl_get($ctx, 'IsActive')) ?>" data-description="<?= tpl_escape(tpl_get($ctx, 'Inventory.Description')) ?>" data-container-label="<?php if (tpl_truthy(tpl_get($ctx, 'Inventory.ContainerNo'))): ?><?= tpl_escape(tpl_get($ctx, 'Inventory.ContainerNo')) ?> · <?= tpl_escape(tpl_container_size_label(tpl_get($ctx, 'Inventory.ContainerSize'))) ?><?php else: ?>LCL · <?= tpl_escape(tpl_number(tpl_get($ctx, 'Inventory.EstimatedVolumeM3'))) ?> m³<?php endif; ?>" data-determination="<?= tpl_escape(tpl_get($ctx, 'Inventory.DeterminationNo')) ?>" data-search="<?= tpl_escape(tpl_lower(tpl_get($ctx, 'Inventory.DeterminationNo'))) ?> <?= tpl_escape(tpl_lower(tpl_get($ctx, 'Inventory.ContainerNo'))) ?> <?= tpl_escape(tpl_lower(tpl_get($ctx, 'Inventory.Description'))) ?>" hidden>
              <input type="checkbox" name="process_ids[]" value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>" data-process-candidate-checkbox disabled>
              <span><strong><?php if (tpl_truthy(tpl_get($ctx, 'Inventory.ContainerNo'))): ?><?= tpl_escape(tpl_get($ctx, 'Inventory.ContainerNo')) ?> · <?= tpl_escape(tpl_get($ctx, 'Inventory.ContainerSize')) ?>'<?php else: ?>LCL · <?= tpl_escape(tpl_number(tpl_get($ctx, 'Inventory.EstimatedVolumeM3'))) ?> m³<?php endif; ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Inventory.Type')) ?> · <?= tpl_escape(tpl_get($ctx, 'Inventory.DeterminationNo')) ?></small><em><?= tpl_escape(tpl_get($ctx, 'Inventory.Description')) ?></em></span><i><?= tpl_escape(tpl_get($ctx, 'StatusLabel')) ?></i>
            </label>
            <?php $ctx = $__parent6; endforeach; endif; ?>
          </div>
          <p class="picker-empty" data-process-picker-empty hidden>Tidak ada barang dengan status yang sesuai untuk action ini.</p>
        </section>

        <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ProcessType'), 'lelang'))): ?>
        <section class="action-field-section auction-schedule-picker" data-auction-schedule-picker hidden>
          <div class="picker-heading"><div><h3>Pilih ND penjadwalan lelang</h3><p>Pilih satu ND, lalu tetapkan hasil setiap kontainer atau barang di dalam jadwal tersebut.</p></div><strong data-auction-schedule-count>0 ND dipilih</strong></div>
          <input type="hidden" name="auction_schedule_no" data-auction-schedule-no disabled>
          <input type="hidden" name="auction_results_json" data-auction-results-json disabled>
          <div class="picker-controls auction-picker-controls">
            <label class="search-field large"><svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/></svg><input type="search" data-auction-schedule-search placeholder="Cari nomor ND penjadwalan lelang…" autocomplete="off"></label>
            <button type="button" class="picker-clear" data-auction-schedule-clear disabled>Kosongkan pencarian</button>
          </div>
          <div class="auction-filter-summary"><span data-auction-schedule-visible><?= tpl_escape(tpl_len(tpl_get($ctx, 'AuctionScheduleGroups'))) ?> ND tersedia</span><small>Pilih satu ND untuk membuka daftar barang di dalamnya.</small></div>
          <div class="auction-schedule-list">
            <?php $__range7 = tpl_iter(tpl_get($ctx, 'AuctionScheduleGroups')); if (count($__range7) > 0): $__parent7 = $ctx; foreach ($__range7 as $__key7 => $__item7): $ctx = $__item7; ?>
            <article class="auction-schedule-card" data-auction-schedule-card data-schedule-no="<?= tpl_escape(tpl_get($ctx, 'DocumentNo')) ?>" data-auction-schedule-search-value="<?= tpl_escape(tpl_lower(tpl_get($ctx, 'DocumentNo'))) ?>">
              <button type="button" class="auction-schedule-button" data-auction-schedule-button><span class="auction-schedule-leading"><i class="auction-schedule-badge">ND</i><span class="auction-schedule-copy"><strong><?= tpl_escape(tpl_get($ctx, 'DocumentNo')) ?></strong><small><?= tpl_escape(tpl_date(tpl_get($ctx, 'DocumentDate'))) ?> · <?= tpl_escape(tpl_len(tpl_get($ctx, 'Processes'))) ?> barang menunggu hasil</small></span></span><span class="auction-schedule-action"><em>Pilih jadwal</em><svg viewBox="0 0 24 24"><path d="m9 18 6-6-6-6"/></svg></span></button>
              <div class="auction-result-list" data-auction-result-list hidden>
                <?php $__range8 = tpl_iter(tpl_get($ctx, 'Processes')); if (count($__range8) > 0): $__parent8 = $ctx; foreach ($__range8 as $__key8 => $__item8): $ctx = $__item8; ?>
                <section class="auction-result-row" data-auction-result-row data-process-id="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>">
                  <header><div><strong><?php if (tpl_truthy(tpl_get($ctx, 'Inventory.ContainerNo'))): ?><?= tpl_escape(tpl_get($ctx, 'Inventory.ContainerNo')) ?> · <?= tpl_escape(tpl_container_size_label(tpl_get($ctx, 'Inventory.ContainerSize'))) ?><?php else: ?>LCL · <?= tpl_escape(tpl_number(tpl_get($ctx, 'Inventory.EstimatedVolumeM3'))) ?> m³<?php endif; ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Inventory.DeterminationNo')) ?> · HTL <?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'HTLValue'))) ?></small></div><p><?= tpl_escape(tpl_get($ctx, 'Inventory.Description')) ?></p></header>
                  <div class="form-grid cols-2"><label>Status hasil <em>*</em><select data-auction-result-outcome disabled><option value="">Pilih hasil</option><option value="laku">Laku</option><option value="tidak_laku">Tidak laku</option></select></label><label data-auction-result-sale hidden>Harga jual <em>*</em><input inputmode="numeric" data-auction-result-sale-input disabled placeholder="Nilai hasil penjualan"></label></div>
                </section>
                <?php $ctx = $__parent8; endforeach; endif; ?>
              </div>
            </article>
            <?php $ctx = $__parent7; endforeach; else: ?><div class="picker-empty static-empty">Belum ada ND penjadwalan lelang yang menunggu penetapan hasil.</div><?php endif; ?>
          </div>
          <p class="picker-empty" data-auction-schedule-empty hidden>Nomor ND penjadwalan tidak ditemukan.</p>
        </section>
        <?php endif; ?>

        <div class="action-fields" data-process-action-fields hidden>
          <section class="action-field-section"><h3>Dokumen action</h3><div class="form-grid cols-2"><label data-document-label>Nomor dokumen <em>*</em><input name="document_no" required disabled></label><label>Tanggal dokumen <em>*</em><input type="date" name="document_date" value="<?= tpl_escape(tpl_isodate(tpl_get($ctx, 'Now'))) ?>" required disabled></label><label class="span-2 document-upload-field">Upload dokumen <span class="optional-mark">Opsional</span><input type="file" name="document_file" accept="application/pdf,image/jpeg,image/png,image/webp,image/gif" disabled><small>PDF atau gambar, maksimal 8 MB. File dapat diunduh kembali melalui timeline.</small></label></div></section>

          <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'ProcessType'), 'lelang'))): ?>
          <section class="action-field-section" data-process-fields-for="kep_htl" hidden><h3>Harga Terendah Lelang per barang</h3><input type="hidden" name="htl_results_json" data-htl-results-json disabled><div class="per-item-value-list" data-htl-editor-list><p class="field-help">Pilih satu atau beberapa barang. Setiap barang akan memperoleh input nilai HTL tersendiri.</p></div><p class="field-help">Nilai HTL memengaruhi barang yang bersangkutan saja dan tidak mengganti nilai penelitian PFPD.</p></section>
          <section class="action-field-section" data-process-fields-for="jadwal_lelang" hidden><h3>Pelaksanaan lelang</h3><div class="form-grid cols-2"><label>Tanggal mulai <em>*</em><input type="date" name="execution_start_date" required disabled></label><label>Tanggal selesai<input type="date" name="execution_end_date" disabled></label></div><p class="field-help">Kosongkan tanggal selesai apabila pelaksanaan hanya satu hari.</p></section>
          <section class="action-field-section" data-process-fields-for="selesai_lelang" hidden><h3>Hasil lelang</h3><p class="field-help">Nomor dan tanggal risalah diterapkan kepada seluruh barang dalam ND penjadwalan yang dipilih.</p></section>
          <section class="action-field-section" data-process-fields-for="lelang_penyesuaian" hidden><h3>Lelang penyesuaian</h3><p class="field-help">Hanya barang berstatus Tidak Laku yang dapat dipilih. Putaran bertambah tanpa penetapan HTL baru.</p></section>
          <section class="action-field-section" data-process-fields-for="alokasi_hasil_lelang" hidden><h3>Alokasi hasil lelang</h3><label>Dialokasikan ke mana <em>*</em><textarea name="allocation_target" rows="3" required disabled placeholder="Isi tujuan atau keterangan alokasi hasil lelang"></textarea></label></section>
          <?php elseif (tpl_truthy(tpl_eq(tpl_get($ctx, 'ProcessType'), 'musnah'))): ?>
          <section class="action-field-section" data-process-fields-for="kep_musnah" hidden><h3>Biaya pemusnahan</h3><label>Biaya musnah <em>*</em><input inputmode="numeric" name="destruction_cost" required disabled placeholder="Biaya dalam rupiah"></label></section>
          <section class="action-field-section" data-process-fields-for="ba_musnah" hidden><h3>Pelaksanaan pemusnahan</h3><label>Biaya musnah aktual <em>*</em><input inputmode="numeric" name="destruction_cost" required disabled placeholder="Biaya aktual dalam rupiah"></label></section>
          <?php else: ?>
          <section class="action-field-section" data-process-fields-for="ba_serah_terima" hidden><h3>Jenis serah terima</h3><label>Jenis <em>*</em><select name="transfer_type" required disabled><option value="">Pilih jenis</option><?php $__range9 = tpl_iter(tpl_get($ctx, 'TransferTypeOptions')); if (count($__range9) > 0): $__parent9 = $ctx; foreach ($__range9 as $__key9 => $__item9): $ctx = $__item9; ?><option value="<?= tpl_escape(tpl_get($ctx, 'Code')) ?>"><?= tpl_escape(tpl_get($ctx, 'Label')) ?></option><?php $ctx = $__parent9; endforeach; endif; ?></select></label></section>
          <?php endif; ?>

          <section class="action-field-section"><h3>Catatan</h3><label>Keterangan tambahan<textarea name="notes" rows="3" placeholder="Opsional"></textarea></label></section>
        </div>
        </section>
      </div>
      <footer class="modal-footer action-modal-footer"><button class="button ghost" type="button" data-close-modal>Batal</button><button class="button primary" type="submit" disabled hidden data-submit-step data-action-submit>Simpan action</button></footer>
    </form>
  </section>
</div>
<?php endif; ?>
<?php endif; ?>

