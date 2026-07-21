<section class="page-intro">
  <div><p><?= tpl_escape(tpl_get($ctx, 'Subtitle')) ?></p></div>
  <div class="intro-actions dashboard-top-controls">
    <form method="get" action="/" class="dashboard-inventory-scope-form">
      <?php if (tpl_truthy(tpl_get($ctx, 'FacilityID'))): ?><input type="hidden" name="tpp" value="<?= tpl_escape(tpl_get($ctx, 'FacilityID')) ?>"><?php endif; ?>
      <label><span>Cakupan inventory</span><select name="inventory_scope" data-auto-submit aria-label="Pilih cakupan angka inventory dashboard"><option value="all_office" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'DashboardInventoryScope'), 'all_office'))): ?>selected<?php endif; ?>>Seluruh cakupan Kantor Tanjung Priok</option><option value="still_tps" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'DashboardInventoryScope'), 'still_tps'))): ?>selected<?php endif; ?>>Masih di TPS</option><option value="all_tpp" <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'DashboardInventoryScope'), 'all_tpp'))): ?>selected<?php endif; ?>>Seluruh TPP</option><?php $__range1 = tpl_iter(tpl_get($ctx, 'Facilities')); if (count($__range1) > 0): $__parent1 = $ctx; foreach ($__range1 as $__key1 => $__item1): $ctx = $__item1; ?><option value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>" <?php if (tpl_truthy(tpl_eq(tpl_get($root, 'DashboardInventoryScope'), tpl_get($ctx, 'ID')))): ?>selected<?php endif; ?>><?= tpl_escape(tpl_get($ctx, 'Name')) ?></option><?php $ctx = $__parent1; endforeach; endif; ?></select></label>
    </form>
    <span class="date-chip"><svg viewBox="0 0 24 24"><path d="M5 4v3M19 4v3M3 9h18M5 6h14a2 2 0 0 1 2 2v12H3V8a2 2 0 0 1 2-2Z"/></svg><?= tpl_escape(tpl_date(tpl_get($ctx, 'Now'))) ?></span>
  </div>
</section>

<section class="kpi-grid">
  <a class="kpi-card primary-card kpi-card-link" href="/inventory" aria-label="Buka seluruh inventory aktif">
    <div class="kpi-icon"><svg viewBox="0 0 24 24"><path d="M4 7.5 12 3l8 4.5v9L12 21l-8-4.5z"/><path d="m4 7.5 8 4.5 8-4.5M12 12v9"/></svg></div>
    <div><span>Total inventory aktif</span><strong><?= tpl_escape(tpl_get($ctx, 'Stats.ActiveTotal')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'DashboardInventoryLabel')) ?></small><small class="kpi-breakdown"><?= tpl_escape(tpl_get($ctx, 'Stats.ActiveSummary.Documents')) ?> dokumen · <?= tpl_escape(tpl_get($ctx, 'Stats.ActiveSummary.FCL')) ?> FCL · <?= tpl_escape(tpl_get($ctx, 'Stats.ActiveSummary.LCL')) ?> LCL</small></div>
    <span class="kpi-card-arrow" aria-hidden="true">→</span>
  </a>
  <article class="kpi-card"><span class="kpi-badge blue">BTD</span><div><span>Barang Tidak Dikuasai</span><strong><?= tpl_escape(tpl_get($ctx, 'Stats.BTDTotal')) ?></strong><small>dalam inventory aktif</small><small class="kpi-breakdown"><?= tpl_escape(tpl_get($ctx, 'Stats.BTDSummary.Documents')) ?> dokumen · <?= tpl_escape(tpl_get($ctx, 'Stats.BTDSummary.FCL')) ?> FCL · <?= tpl_escape(tpl_get($ctx, 'Stats.BTDSummary.LCL')) ?> LCL</small></div></article>
  <article class="kpi-card"><span class="kpi-badge amber">BDN</span><div><span>Barang Dikuasai Negara</span><strong><?= tpl_escape(tpl_get($ctx, 'Stats.BDNTotal')) ?></strong><small>dalam inventory aktif</small><small class="kpi-breakdown"><?= tpl_escape(tpl_get($ctx, 'Stats.BDNSummary.Documents')) ?> dokumen · <?= tpl_escape(tpl_get($ctx, 'Stats.BDNSummary.FCL')) ?> FCL · <?= tpl_escape(tpl_get($ctx, 'Stats.BDNSummary.LCL')) ?> LCL</small></div></article>
  <article class="kpi-card"><span class="kpi-badge violet">BMMN</span><div><span>Barang Milik Negara</span><strong><?= tpl_escape(tpl_get($ctx, 'Stats.BMMNTotal')) ?></strong><small>siap/proses peruntukan</small><small class="kpi-breakdown"><?= tpl_escape(tpl_get($ctx, 'Stats.BMMNSummary.Documents')) ?> dokumen · <?= tpl_escape(tpl_get($ctx, 'Stats.BMMNSummary.FCL')) ?> FCL · <?= tpl_escape(tpl_get($ctx, 'Stats.BMMNSummary.LCL')) ?> LCL</small></div></article>
  <article class="kpi-card"><span class="kpi-badge green">TIT</span><div><span>Barang Titipan</span><strong><?= tpl_escape(tpl_get($ctx, 'Stats.TitipanTotal')) ?></strong><small>dalam inventory aktif</small><small class="kpi-breakdown"><?= tpl_escape(tpl_get($ctx, 'Stats.TitipanSummary.Documents')) ?> dokumen · <?= tpl_escape(tpl_get($ctx, 'Stats.TitipanSummary.FCL')) ?> FCL · <?= tpl_escape(tpl_get($ctx, 'Stats.TitipanSummary.LCL')) ?> LCL</small></div></article>
</section>

<section class="occupancy-panel panel">
  <header class="panel-header occupancy-header">
    <div><h2>YOR & SOR</h2><p>YOR dihitung dalam TEU (setara peti kemas 20 kaki), sedangkan SOR dihitung dalam m³: <?= tpl_escape(tpl_get($ctx, 'DashboardScope')) ?></p></div>
    <div class="occupancy-header-actions">
      <form method="get" action="/" class="dashboard-scope-form"><input type="hidden" name="inventory_scope" value="<?= tpl_escape(tpl_get($ctx, 'DashboardInventoryScope')) ?>"><label><span>Tampilkan detail</span><select name="tpp" data-auto-submit><option value="">Gabungan seluruh TPP</option><?php $__range2 = tpl_iter(tpl_get($ctx, 'Facilities')); if (count($__range2) > 0): $__parent2 = $ctx; foreach ($__range2 as $__key2 => $__item2): $ctx = $__item2; ?><option value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>" <?php if (tpl_truthy(tpl_eq(tpl_get($root, 'FacilityID'), tpl_get($ctx, 'ID')))): ?>selected<?php endif; ?>><?= tpl_escape(tpl_get($ctx, 'Name')) ?></option><?php $ctx = $__parent2; endforeach; endif; ?></select></label></form>
      <?php if (tpl_truthy(tpl_get($ctx, 'CanEditCapacity'))): ?><button class="button secondary compact" type="button" data-open-capacity-editor data-facility-id="<?= tpl_escape(tpl_get($ctx, 'FacilityID')) ?>"><svg viewBox="0 0 24 24"><path d="M4 20h4l11-11-4-4L4 16v4ZM13 7l4 4"/></svg>Edit kapasitas</button><?php endif; ?>
    </div>
  </header>
  <div class="occupancy-grid">
    <article class="occupancy-card yard"><div class="occupancy-title"><span>YOR</span><div><strong>Yard Occupancy Ratio</strong><small><?= tpl_escape(tpl_get($ctx, 'DashboardScope')) ?></small></div></div><div class="occupancy-value"><strong><?= tpl_escape(tpl_percent(tpl_get($ctx, 'DashboardOccupancy.YardUsed'), tpl_get($ctx, 'DashboardOccupancy.YardCapacity'))) ?>%</strong><span><?= tpl_escape(tpl_number(tpl_get($ctx, 'DashboardOccupancy.YardUsed'))) ?> / <?= tpl_escape(tpl_number(tpl_get($ctx, 'DashboardOccupancy.YardCapacity'))) ?> TEU terpakai</span></div><progress value="<?= tpl_escape(tpl_get($ctx, 'DashboardOccupancy.YardUsed')) ?>" max="<?= tpl_escape(tpl_get($ctx, 'DashboardOccupancy.YardCapacity')) ?>"><?= tpl_escape(tpl_percent(tpl_get($ctx, 'DashboardOccupancy.YardUsed'), tpl_get($ctx, 'DashboardOccupancy.YardCapacity'))) ?>%</progress></article>
    <article class="occupancy-card shed"><div class="occupancy-title"><span>SOR</span><div><strong>Shed Occupancy Ratio</strong><small><?= tpl_escape(tpl_get($ctx, 'DashboardScope')) ?></small></div></div><div class="occupancy-value"><strong><?= tpl_escape(tpl_percent(tpl_get($ctx, 'DashboardOccupancy.ShedUsed'), tpl_get($ctx, 'DashboardOccupancy.ShedCapacity'))) ?>%</strong><span><?= tpl_escape(tpl_number(tpl_get($ctx, 'DashboardOccupancy.ShedUsed'))) ?> / <?= tpl_escape(tpl_number(tpl_get($ctx, 'DashboardOccupancy.ShedCapacity'))) ?> m³ terpakai</span></div><progress value="<?= tpl_escape(tpl_get($ctx, 'DashboardOccupancy.ShedUsed')) ?>" max="<?= tpl_escape(tpl_get($ctx, 'DashboardOccupancy.ShedCapacity')) ?>"><?= tpl_escape(tpl_percent(tpl_get($ctx, 'DashboardOccupancy.ShedUsed'), tpl_get($ctx, 'DashboardOccupancy.ShedCapacity'))) ?>%</progress></article>
  </div>
</section>

<section class="process-strip">
  <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'auction.view'))): ?><button class="process-summary process-summary-button" type="button" data-open-process-dashboard="lelang"><span class="process-symbol violet"><svg viewBox="0 0 24 24"><path d="m14 5 5 5M12.5 6.5l5 5M4 20l7-7M9 4l11 11M3 21h7"/></svg></span><span class="process-summary-copy"><small>Lelang aktif</small><strong><?= tpl_escape(tpl_get($ctx, 'Stats.AuctionActive')) ?></strong></span><span class="process-summary-link">Lihat dashboard</span></button><?php else: ?><div class="process-summary"><span class="process-symbol violet"><svg viewBox="0 0 24 24"><path d="m14 5 5 5M12.5 6.5l5 5M4 20l7-7M9 4l11 11M3 21h7"/></svg></span><div><small>Lelang aktif</small><strong><?= tpl_escape(tpl_get($ctx, 'Stats.AuctionActive')) ?></strong></div></div><?php endif; ?>
  <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'destruction.view'))): ?><button class="process-summary process-summary-button" type="button" data-open-process-dashboard="musnah"><span class="process-symbol red"><svg viewBox="0 0 24 24"><path d="M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14"/></svg></span><span class="process-summary-copy"><small>Pemusnahan aktif</small><strong><?= tpl_escape(tpl_get($ctx, 'Stats.DestructionActive')) ?></strong></span><span class="process-summary-link">Lihat dashboard</span></button><?php else: ?><div class="process-summary"><span class="process-symbol red"><svg viewBox="0 0 24 24"><path d="M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14"/></svg></span><div><small>Pemusnahan aktif</small><strong><?= tpl_escape(tpl_get($ctx, 'Stats.DestructionActive')) ?></strong></div></div><?php endif; ?>
  <?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'grant.view'))): ?><button class="process-summary process-summary-button" type="button" data-open-process-dashboard="hibah"><span class="process-symbol teal"><svg viewBox="0 0 24 24"><path d="M20.8 8.4c0 5.2-8.8 10.1-8.8 10.1S3.2 13.6 3.2 8.4A4.4 4.4 0 0 1 12 8a4.4 4.4 0 0 1 8.8.4Z"/></svg></span><span class="process-summary-copy"><small>Hibah/PSP aktif</small><strong><?= tpl_escape(tpl_get($ctx, 'Stats.GrantActive')) ?></strong></span><span class="process-summary-link">Lihat dashboard</span></button><?php else: ?><div class="process-summary"><span class="process-symbol teal"><svg viewBox="0 0 24 24"><path d="M20.8 8.4c0 5.2-8.8 10.1-8.8 10.1S3.2 13.6 3.2 8.4A4.4 4.4 0 0 1 12 8a4.4 4.4 0 0 1 8.8.4Z"/></svg></span><div><small>Hibah/PSP aktif</small><strong><?= tpl_escape(tpl_get($ctx, 'Stats.GrantActive')) ?></strong></div></div><?php endif; ?>
  <button class="process-summary process-summary-button performance-summary-button" type="button" data-open-performance-dashboard><span class="process-symbol green"><svg viewBox="0 0 24 24"><path d="M4 19V9M10 19V5M16 19v-7M22 19V2M2 19h22"/></svg></span><span class="process-summary-copy"><small>Performa kinerja</small><strong><?= tpl_escape(tpl_get($ctx, 'Performance.TotalCompleted')) ?></strong><em><?= tpl_escape(tpl_get($ctx, 'Performance.PeriodLabel')) ?></em></span><span class="process-summary-link">Lihat performa</span></button>
</section>

<section class="dashboard-grid">
  <article class="panel facility-panel">
    <header class="panel-header"><div><h2>Detail per TPP</h2><p>Inventory aktif yang berada di masing-masing TPP; tidak dipengaruhi pilihan cakupan KPI</p></div><a class="text-link" href="/inventory">Lihat inventory →</a></header>
    <div class="facility-table" role="table">
      <div class="facility-row occupancy-row facility-head" role="row"><span>TPP</span><span>BTD</span><span>BDN</span><span>BMMN</span><span>Titipan</span><span>Total</span><span>YOR</span><span>SOR</span></div>
      <?php $__range3 = tpl_iter(tpl_get($ctx, 'DashboardRows')); if (count($__range3) > 0): $__parent3 = $ctx; foreach ($__range3 as $__key3 => $__item3): $ctx = $__item3; ?>
      <div class="facility-row occupancy-row" role="row"><span><i><?= tpl_escape(tpl_initials(tpl_get($ctx, 'FacilityName'))) ?></i><?= tpl_escape(tpl_get($ctx, 'FacilityName')) ?></span><span><?= tpl_escape(tpl_get($ctx, 'BTD')) ?></span><span><?= tpl_escape(tpl_get($ctx, 'BDN')) ?></span><span><?= tpl_escape(tpl_get($ctx, 'BMMN')) ?></span><span><?= tpl_escape(tpl_get($ctx, 'Titipan')) ?></span><strong><?= tpl_escape(tpl_get($ctx, 'Total')) ?></strong><span><?= tpl_escape(tpl_percent(tpl_get($ctx, 'YardUsed'), tpl_get($ctx, 'YardCapacity'))) ?>%</span><span><?= tpl_escape(tpl_percent(tpl_get($ctx, 'ShedUsed'), tpl_get($ctx, 'ShedCapacity'))) ?>%</span></div>
      <?php $ctx = $__parent3; endforeach; endif; ?>
    </div>
  </article>

  <article class="panel activity-panel">
    <header class="panel-header"><div><h2>Aktivitas terbaru</h2><p>Pembaruan terakhir dari seluruh proses</p></div></header>
    <div class="activity-list">
      <?php $__range4 = tpl_iter(tpl_get($ctx, 'Stats.RecentEvents')); if (count($__range4) > 0): $__parent4 = $ctx; foreach ($__range4 as $__key4 => $__item4): $ctx = $__item4; ?>
      <div class="activity-item"><span class="activity-dot <?= tpl_escape(tpl_status_tone(tpl_get($ctx, 'Code'))) ?>"></span><div><strong><?= tpl_escape(tpl_get($ctx, 'Label')) ?></strong><p><?php if (tpl_truthy(tpl_get($ctx, 'DocumentNo'))): ?><?= tpl_escape(tpl_get($ctx, 'DocumentNo')) ?><?php else: ?><?= tpl_escape(tpl_get($ctx, 'Notes')) ?><?php endif; ?></p><small><?= tpl_escape(tpl_datetime(tpl_get($ctx, 'CreatedAt'))) ?> · <?= tpl_escape(tpl_get($ctx, 'Actor')) ?></small></div></div>
      <?php $ctx = $__parent4; endforeach; else: ?><div class="empty-mini">Belum ada aktivitas.</div><?php endif; ?>
    </div>
  </article>
</section>

<section class="panel attention-panel">
  <header class="panel-header"><div><h2>Perlu perhatian</h2><p>Barang aktif berusia 45 hari atau lebih sejak penetapan</p></div><span class="count-chip"><?= tpl_escape(tpl_len(tpl_get($ctx, 'Stats.AttentionItems'))) ?> prioritas</span></header>
  <div class="table-wrap">
    <table class="data-table compact-table">
      <thead><tr><th>Referensi</th><th>Kontainer & barang</th><th>TPP</th><th>Umur</th><th>Status terakhir</th><th></th></tr></thead>
      <tbody>
      <?php $__range5 = tpl_iter(tpl_get($ctx, 'Stats.AttentionItems')); if (count($__range5) > 0): $__parent5 = $ctx; foreach ($__range5 as $__key5 => $__item5): $ctx = $__item5; ?>
        <tr><td><strong><?= tpl_escape(tpl_get($ctx, 'DeterminationNo')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Type')) ?></small></td><td><strong><?php if (tpl_truthy(tpl_get($ctx, 'ContainerNo'))): ?><?= tpl_escape(tpl_get($ctx, 'ContainerNo')) ?> · <?= tpl_escape(tpl_get($ctx, 'ContainerSize')) ?>'<?php else: ?>LCL · <?= tpl_escape(tpl_number(tpl_get($ctx, 'EstimatedVolumeM3'))) ?> m³<?php endif; ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Description')) ?></small></td><td><?php if (tpl_truthy(tpl_get($ctx, 'FacilityName'))): ?><?= tpl_escape(tpl_get($ctx, 'FacilityName')) ?><?php else: ?><?= tpl_escape(tpl_get($ctx, 'OriginWarehouse')) ?><?php endif; ?></td><td><span class="age-pill danger"><?= tpl_escape(tpl_age($ctx, tpl_get($root, 'Now'))) ?> hari</span></td><td><button class="status-button <?= tpl_escape(tpl_status_tone(tpl_get($ctx, 'StatusCode'))) ?>" type="button" data-timeline-url="/api/inventory/<?= tpl_escape(tpl_get($ctx, 'ID')) ?>/timeline"><?= tpl_escape(tpl_get($ctx, 'StatusLabel')) ?><svg viewBox="0 0 24 24"><path d="m9 18 6-6-6-6"/></svg></button></td><td><a class="row-link" href="/inventory?q=<?= tpl_escape(tpl_get($ctx, 'ContainerNo')) ?>">Buka</a></td></tr>
      <?php $ctx = $__parent5; endforeach; else: ?><tr><td colspan="6"><div class="empty-state small">Tidak ada barang yang melewati batas perhatian.</div></td></tr><?php endif; ?>
      </tbody>
    </table>
  </div>
</section>

<?php if (tpl_truthy(tpl_get($ctx, 'CanEditCapacity'))): ?>
<div class="modal" id="capacity-edit-modal" role="dialog" aria-modal="true" aria-labelledby="capacity-edit-title" hidden>
  <div class="modal-backdrop" data-close-modal></div>
  <section class="modal-panel modal-panel-medium">
    <header class="modal-header"><div><p class="eyebrow">Pengaturan TPP</p><h2 id="capacity-edit-title">Edit kapasitas YOR & SOR</h2><p>YOR menggunakan satuan TEU atau ekuivalen peti kemas 20 kaki. SOR menggunakan meter kubik.</p></div><button class="icon-button" type="button" data-close-modal aria-label="Tutup"><svg viewBox="0 0 24 24"><path d="m6 6 12 12M18 6 6 18"/></svg></button></header>
    <form class="modal-form" method="post" action="" data-capacity-form>
      <input type="hidden" name="_csrf" value="<?= tpl_escape(tpl_get($ctx, 'CSRF')) ?>">
      <fieldset><legend>TPP dan kapasitas maksimum</legend><div class="form-grid cols-2">
        <label class="span-2">Nama TPP <em>*</em><select data-capacity-facility required><option value="">Pilih TPP</option><?php $__range6 = tpl_iter(tpl_get($ctx, 'Facilities')); if (count($__range6) > 0): $__parent6 = $ctx; foreach ($__range6 as $__key6 => $__item6): $ctx = $__item6; ?><option value="<?= tpl_escape(tpl_get($ctx, 'ID')) ?>" data-yard-capacity="<?= tpl_escape(tpl_get($ctx, 'YardCapacity')) ?>" data-shed-capacity="<?= tpl_escape(tpl_get($ctx, 'ShedCapacity')) ?>"><?= tpl_escape(tpl_get($ctx, 'Name')) ?></option><?php $ctx = $__parent6; endforeach; endif; ?></select></label>
        <label>Kapasitas YOR <em>*</em><input type="number" min="0" step="0.01" name="yard_capacity" data-yard-capacity-input required><small class="field-inline-help">TEU (setara peti kemas 20')</small></label>
        <label>Kapasitas SOR <em>*</em><input type="number" min="0" step="0.01" name="shed_capacity" data-shed-capacity-input required><small class="field-inline-help">Meter kubik (m³)</small></label>
      </div></fieldset>
      <p class="field-help">Pemakaian YOR dihitung otomatis dari ukuran kontainer FCL aktif. Pemakaian SOR dihitung dari perkiraan volume LCL aktif yang sudah berada di TPP.</p>
      <footer class="modal-footer"><button class="button ghost" type="button" data-close-modal>Batal</button><button class="button primary" type="submit">Simpan kapasitas</button></footer>
    </form>
  </section>
</div>
<?php endif; ?>

<div class="modal performance-dashboard-modal" id="performance-dashboard-modal" role="dialog" aria-modal="true" aria-labelledby="performance-dashboard-title" data-auto-open="<?php if (tpl_truthy(tpl_get($ctx, 'PerformanceOpen'))): ?>true<?php else: ?>false<?php endif; ?>" hidden>
  <div class="modal-backdrop" data-close-modal></div>
  <section class="modal-panel modal-panel-dashboard">
    <header class="modal-header"><div><p class="eyebrow">Dashboard kinerja</p><h2 id="performance-dashboard-title">Performa Kinerja</h2><p>Jumlah penyelesaian dan rata-rata waktu layanan berdasarkan tanggal dokumen penyelesaian.</p></div><button class="icon-button" type="button" data-close-modal aria-label="Tutup"><svg viewBox="0 0 24 24"><path d="m6 6 12 12M18 6 6 18"/></svg></button></header>
    <div class="process-dashboard-modal-scroll performance-dashboard-scroll">
      <form class="performance-filter-form" method="get" action="/">
        <input type="hidden" name="performance" value="1">
        <input type="hidden" name="inventory_scope" value="<?= tpl_escape(tpl_get($ctx, 'DashboardInventoryScope')) ?>">
        <?php if (tpl_truthy(tpl_get($ctx, 'FacilityID'))): ?><input type="hidden" name="tpp" value="<?= tpl_escape(tpl_get($ctx, 'FacilityID')) ?>"><?php endif; ?>
        <div><strong>Periode pengukuran</strong><small>Default menampilkan satu tahun kalender berjalan. Rentang dapat diubah sesuai kebutuhan.</small></div>
        <label>Dari tanggal<input type="date" name="performance_from" value="<?= tpl_escape(tpl_get($ctx, 'Performance.DateFromInput')) ?>" required></label>
        <label>Sampai tanggal<input type="date" name="performance_to" value="<?= tpl_escape(tpl_get($ctx, 'Performance.DateToInput')) ?>" required></label>
        <button class="button primary compact" type="submit">Terapkan filter</button>
      </form>
      <section class="performance-overview">
        <header><div><p class="eyebrow"><?= tpl_escape(tpl_get($ctx, 'Performance.PeriodLabel')) ?></p><h3>Ringkasan performa tahapan</h3></div><span><?= tpl_escape(tpl_get($ctx, 'Performance.TotalCompleted')) ?> penyelesaian</span></header>
        <div class="performance-metric-grid">
          <?php $__range7 = tpl_iter(tpl_get($ctx, 'Performance.Metrics')); if (count($__range7) > 0): $__parent7 = $ctx; foreach ($__range7 as $__key7 => $__item7): $ctx = $__item7; ?>
          <article class="performance-metric-card"><div><span><?= tpl_escape(tpl_get($ctx, 'Label')) ?></span><strong><?= tpl_escape(tpl_get($ctx, 'Count')) ?></strong><small>proses/dokumen selesai</small></div><div class="performance-duration"><span>Waktu rata-rata</span><strong><?= tpl_escape(tpl_performance_duration(tpl_get($ctx, 'AverageHours'), tpl_get($ctx, 'DurationSamples'))) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'DurationSamples')) ?> durasi valid</small></div><p><?= tpl_escape(tpl_get($ctx, 'Description')) ?></p></article>
          <?php $ctx = $__parent7; endforeach; endif; ?>
        </div>
        <aside class="performance-method-note"><strong>Dasar penghitungan</strong><p>Lelang, musnah, hibah/PSP, cacah, dan konversi BMMN dihitung dari penetapan awal BTD/BDN. Untuk barang yang sudah menjadi BMMN, sistem tetap memakai tanggal dokumen asal sebelum konversi. Penilaian PFPD dihitung dari tanggal request penelitian sampai tanggal penilaian.</p></aside>
      </section>
    </div>
    <footer class="modal-footer"><button class="button ghost" type="button" data-close-modal>Tutup</button><?php if (tpl_truthy(tpl_can(tpl_get($ctx, 'User'), 'reports.view'))): ?><a class="button secondary" href="<?= tpl_escape(tpl_get($ctx, 'Performance.ExportURL')) ?>"><svg viewBox="0 0 24 24"><path d="M12 3v12M7 10l5 5 5-5M5 21h14"/></svg>Unduh Excel</a><a class="button primary" href="/pelaporan?preset=performance&amp;date_from=<?= tpl_escape(tpl_get($ctx, 'Performance.DateFromInput')) ?>&amp;date_to=<?= tpl_escape(tpl_get($ctx, 'Performance.DateToInput')) ?>">Buka di pelaporan</a><?php endif; ?></footer>
  </section>
</div>

<?php $__range8 = tpl_iter(tpl_get($ctx, 'ProcessModals')); if (count($__range8) > 0): $__parent8 = $ctx; foreach ($__range8 as $__key8 => $__item8): $ctx = $__item8; ?>
<?php $modal = $ctx; ?>
<div class="modal process-dashboard-modal" id="process-dashboard-<?= tpl_escape(tpl_get($ctx, 'Type')) ?>" role="dialog" aria-modal="true" aria-labelledby="process-dashboard-title-<?= tpl_escape(tpl_get($ctx, 'Type')) ?>" hidden>
  <div class="modal-backdrop" data-close-modal></div>
  <section class="modal-panel modal-panel-dashboard">
    <header class="modal-header"><div><p class="eyebrow">Ringkasan proses</p><h2 id="process-dashboard-title-<?= tpl_escape(tpl_get($ctx, 'Type')) ?>"><?= tpl_escape(tpl_get($ctx, 'Title')) ?></h2><p>Tren tahun <?= tpl_escape(tpl_get($ctx, 'Dashboard.Year')) ?>, termasuk barang yang sudah masuk history.</p></div><button class="icon-button" type="button" data-close-modal aria-label="Tutup"><svg viewBox="0 0 24 24"><path d="m6 6 12 12M18 6 6 18"/></svg></button></header>
    <div class="process-dashboard-modal-scroll">
      <section class="panel process-dashboard embedded-process-dashboard">
        <header class="process-dashboard-header"><div><p class="eyebrow">Dashboard <?= tpl_escape(tpl_get($ctx, 'Singular')) ?></p><h2>Ringkasan dan tren <?= tpl_escape(tpl_get($ctx, 'Dashboard.Year')) ?></h2><small>Grafik dihitung berdasarkan bulan proses dibuat.</small></div><span><?= tpl_escape(tpl_get($ctx, 'Dashboard.StartedThisYear')) ?> proses dimulai · <?= tpl_escape(tpl_get($ctx, 'Dashboard.CompletedThisYear')) ?> selesai tahun ini</span></header>
        <div class="process-dashboard-kpis">
          <?php if (tpl_truthy(tpl_eq(tpl_get($ctx, 'Type'), 'lelang'))): ?>
          <article><span>Proses dimulai tahun ini</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.StartedThisYear')) ?></strong><small>berdasarkan tanggal mulai</small></article><article><span>Selesai tahun ini</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.CompletedThisYear')) ?></strong><small>berdasarkan tanggal penyelesaian</small></article><article><span>Lelang aktif</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.Active')) ?></strong><small>masih membutuhkan tindak lanjut</small></article><article><span>Total nilai jual</span><strong class="compact-money"><?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'Dashboard.TotalSaleValue'))) ?></strong><small>hasil lelang tercatat</small></article>
          <?php elseif (tpl_truthy(tpl_eq(tpl_get($ctx, 'Type'), 'musnah'))): ?>
          <article><span>Proses dimulai tahun ini</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.StartedThisYear')) ?></strong><small>berdasarkan KEP Musnah</small></article><article><span>Selesai tahun ini</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.CompletedThisYear')) ?></strong><small>berdasarkan BA Musnah</small></article><article><span>Pemusnahan aktif</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.Active')) ?></strong><small>masih berjalan</small></article><article><span>Total biaya musnah</span><strong class="compact-money"><?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'Dashboard.TotalCost'))) ?></strong><small>biaya pemusnahan tercatat</small></article>
          <?php else: ?>
          <article><span>Proses dimulai tahun ini</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.StartedThisYear')) ?></strong><small>Hibah dan PSP</small></article><article><span>Selesai tahun ini</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.CompletedThisYear')) ?></strong><small>serah terima selesai</small></article><article><span>Proses aktif</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.Active')) ?></strong><small>belum selesai</small></article><article><span>Barang dihibahkan</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.TotalGrant')) ?></strong><small>jenis Hibah</small></article><article><span>Barang PSP</span><strong><?= tpl_escape(tpl_get($ctx, 'Dashboard.TotalPSP')) ?></strong><small>penetapan status penggunaan</small></article>
          <?php endif; ?>
        </div>
        <div class="process-chart-scroll"><div class="process-chart" role="img" aria-label="Grafik bulanan <?= tpl_escape(tpl_get($ctx, 'Singular')) ?> tahun <?= tpl_escape(tpl_get($ctx, 'Dashboard.Year')) ?>"><?php $__range9 = tpl_iter(tpl_get($ctx, 'Dashboard.Chart')); if (count($__range9) > 0): $__parent9 = $ctx; foreach ($__range9 as $__key9 => $__item9): $ctx = $__item9; ?><article class="chart-month"><header><strong><?= tpl_escape(tpl_get($ctx, 'Label')) ?></strong><small><?= tpl_escape(tpl_get($ctx, 'Count')) ?> proses</small></header><?php if (tpl_truthy(tpl_eq(tpl_get($modal, 'Type'), 'lelang'))): ?><div class="chart-metric goods"><span>Nilai barang</span><progress value="<?= tpl_escape(tpl_get($ctx, 'GoodsValue')) ?>" max="<?= tpl_escape(tpl_get($modal, 'Dashboard.MaxMoney')) ?>"></progress><em><?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'GoodsValue'))) ?></em></div><div class="chart-metric htl"><span>HTL</span><progress value="<?= tpl_escape(tpl_get($ctx, 'HTLValue')) ?>" max="<?= tpl_escape(tpl_get($modal, 'Dashboard.MaxMoney')) ?>"></progress><em><?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'HTLValue'))) ?></em></div><div class="chart-metric sale"><span>Nilai jual</span><progress value="<?= tpl_escape(tpl_get($ctx, 'SaleValue')) ?>" max="<?= tpl_escape(tpl_get($modal, 'Dashboard.MaxMoney')) ?>"></progress><em><?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'SaleValue'))) ?></em></div><?php elseif (tpl_truthy(tpl_eq(tpl_get($modal, 'Type'), 'musnah'))): ?><div class="chart-metric count"><span>Jumlah barang</span><progress value="<?= tpl_escape(tpl_get($ctx, 'Count')) ?>" max="<?= tpl_escape(tpl_get($modal, 'Dashboard.MaxCount')) ?>"></progress><em><?= tpl_escape(tpl_get($ctx, 'Count')) ?></em></div><div class="chart-metric cost"><span>Biaya musnah</span><progress value="<?= tpl_escape(tpl_get($ctx, 'Cost')) ?>" max="<?= tpl_escape(tpl_get($modal, 'Dashboard.MaxMoney')) ?>"></progress><em><?= tpl_escape(tpl_rupiah(tpl_get($ctx, 'Cost'))) ?></em></div><?php else: ?><div class="chart-metric grant"><span>Hibah</span><progress value="<?= tpl_escape(tpl_get($ctx, 'Grant')) ?>" max="<?= tpl_escape(tpl_get($modal, 'Dashboard.MaxCount')) ?>"></progress><em><?= tpl_escape(tpl_get($ctx, 'Grant')) ?></em></div><div class="chart-metric psp"><span>PSP</span><progress value="<?= tpl_escape(tpl_get($ctx, 'PSP')) ?>" max="<?= tpl_escape(tpl_get($modal, 'Dashboard.MaxCount')) ?>"></progress><em><?= tpl_escape(tpl_get($ctx, 'PSP')) ?></em></div><?php endif; ?></article><?php $ctx = $__parent9; endforeach; endif; ?></div></div>
      </section>
    </div>
    <footer class="modal-footer"><button class="button ghost" type="button" data-close-modal>Tutup</button><a class="button primary" href="<?= tpl_escape(tpl_get($ctx, 'URL')) ?>">Buka daftar proses</a></footer>
  </section>
</div>
<?php $ctx = $__parent8; endforeach; endif; ?>
