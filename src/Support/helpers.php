<?php
declare(strict_types=1);


if (!function_exists('mb_strlen')) { function mb_strlen(string $value, ?string $encoding = null): int { return strlen($value); } }
if (!function_exists('mb_strtolower')) { function mb_strtolower(string $value, ?string $encoding = null): string { return strtolower($value); } }
if (!function_exists('mb_strtoupper')) { function mb_strtoupper(string $value, ?string $encoding = null): string { return strtoupper($value); } }
if (!function_exists('mb_substr')) { function mb_substr(string $value, int $offset, ?int $length = null, ?string $encoding = null): string { return $length === null ? substr($value, $offset) : substr($value, $offset, $length); } }


function tpl_key_candidates(string $key): array
{
    // Convert PascalCase/camelCase while preserving acronym groups:
    // BLNo -> bl_no, PFPDRequired -> pfpd_required, MIMEType -> mime_type.
    $snake = (string) preg_replace('/([A-Z]+)([A-Z][a-z])/', '$1_$2', $key);
    $snake = (string) preg_replace('/([a-z0-9])([A-Z])/', '$1_$2', $snake);
    $snake = strtolower(str_replace('-', '_', $snake));
    $camel = lcfirst(str_replace(' ', '', ucwords(str_replace(['_', '-'], ' ', $key))));
    $aliases = match ($key) {
        // Several domain structs intentionally expose the generic field "Type"
        // while Supabase stores a table-specific JSON key.
        'Type' => ['item_type', 'disposition_type', 'reconciliation_type'],
        default => [],
    };
    return array_values(array_unique(array_merge([$key, lcfirst($key), $snake, $camel, strtolower($key)], $aliases)));
}

function tpl_get(mixed $value, string $path, mixed $default = ''): mixed
{
    if ($path === '') return $value;
    foreach (explode('.', $path) as $segment) {
        if ($value === null) return $default;
        $found = false;
        if (is_array($value)) {
            foreach (tpl_key_candidates($segment) as $key) {
                if (array_key_exists($key, $value)) { $value = $value[$key]; $found = true; break; }
            }
        } elseif (is_object($value)) {
            foreach (tpl_key_candidates($segment) as $key) {
                if (isset($value->{$key}) || property_exists($value, $key)) { $value = $value->{$key}; $found = true; break; }
                $getter = 'get'.ucfirst($key);
                if (method_exists($value, $getter)) { $value = $value->{$getter}(); $found = true; break; }
            }
        }
        if (!$found) return $default;
    }
    return $value;
}

function tpl_escape(mixed $value): string
{
    if ($value === null) return '';
    if (is_bool($value)) return $value ? 'true' : 'false';
    if (is_array($value) || is_object($value)) $value = json_encode($value, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    return htmlspecialchars((string)$value, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
}
function tpl_truthy(mixed $v): bool { return !($v === null || $v === false || $v === '' || $v === 0 || $v === 0.0 || $v === []); }
function tpl_eq(mixed $a, mixed $b): bool { return (string)$a === (string)$b; }
function tpl_ne(mixed $a, mixed $b): bool { return !tpl_eq($a, $b); }
function tpl_gt(mixed $a, mixed $b): bool { return $a > $b; }
function tpl_ge(mixed $a, mixed $b): bool { return $a >= $b; }
function tpl_lt(mixed $a, mixed $b): bool { return $a < $b; }
function tpl_le(mixed $a, mixed $b): bool { return $a <= $b; }
function tpl_not(mixed $a): bool { return !tpl_truthy($a); }
function tpl_and(mixed $a, mixed $b): bool { return tpl_truthy($a) && tpl_truthy($b); }
function tpl_or(mixed $a, mixed $b): bool { return tpl_truthy($a) || tpl_truthy($b); }
function tpl_iter(mixed $v): array { if ($v instanceof Traversable) return iterator_to_array($v); return is_array($v) ? $v : []; }
function tpl_index(mixed $v, mixed $index): mixed { return is_array($v) && array_key_exists((int)$index, $v) ? $v[(int)$index] : null; }
function tpl_len(mixed $v): int { return is_countable($v) ? count($v) : (is_string($v) ? mb_strlen($v) : 0); }

function tpl_date_obj(mixed $value): ?DateTimeImmutable
{
    if ($value instanceof DateTimeImmutable) return $value;
    if ($value instanceof DateTimeInterface) return DateTimeImmutable::createFromInterface($value);
    if (!is_string($value) || trim($value) === '' || str_starts_with($value, '0001-01-01')) return null;
    try { return (new DateTimeImmutable($value))->setTimezone(new DateTimeZone('Asia/Jakarta')); } catch (Throwable) { return null; }
}
function tpl_date(mixed $v): string { return ($d=tpl_date_obj($v)) ? $d->format('d/m/Y') : '—'; }
function tpl_datetime(mixed $v): string { return ($d=tpl_date_obj($v)) ? $d->format('d/m/Y H:i') : '—'; }
function tpl_isodate(mixed $v): string { return ($d=tpl_date_obj($v)) ? $d->format('Y-m-d') : ''; }
function tpl_number(mixed $v): string { $n=(float)$v; return rtrim(rtrim(number_format($n, 2, ',', '.'), '0'), ','); }
function tpl_money(mixed $v): string { return 'Rp '.number_format((float)$v, 0, ',', '.'); }
function tpl_rupiah(mixed $v): string { return tpl_money($v); }
function tpl_lower(mixed $v): string { return mb_strtolower((string)$v); }
function tpl_initials(mixed $v): string { $parts=preg_split('/\s+/',trim((string)$v)) ?: []; return mb_strtoupper(mb_substr($parts[0]??'L',0,1).mb_substr($parts[1]??'',0,1)); }
function tpl_age(mixed $item, mixed $now): int { $start=tpl_get($item,'DeterminationDate') ?: tpl_get($item,'CreatedAt'); $a=tpl_date_obj($start); $b=tpl_date_obj($now) ?: new DateTimeImmutable(); return $a ? max(0,(int)$a->diff($b)->format('%a')) : 0; }
function tpl_percent(mixed $used, mixed $capacity): string { $c=(float)$capacity; return $c <= 0 ? '0' : number_format(min(100,max(0,(float)$used/$c*100)),1,',','.'); }
function tpl_performance_duration(mixed $hours, mixed $samples): string { if ((int)$samples <= 0) return '—'; $h=(float)$hours; if ($h<24) return tpl_number($h).' jam'; return tpl_number($h/24).' hari'; }
function tpl_container_size_label(mixed $size): string { return match(strtoupper(trim((string)$size))) {'20'=>"20'",'40'=>"40'",'40HC'=>"40' HC",'45','45HC'=>"45' HC",default=>'—'}; }
function tpl_status_tone(mixed $code): string { $c=(string)$code; if (in_array($c,['laku','ba_musnah','ba_serah_terima','pengeluaran_barang','persetujuan_peruntukan_bmmn'],true)) return 'success'; if (in_array($c,['tidak_laku','rekonsiliasi_tidak_ditemukan'],true)) return 'danger'; if (in_array($c,['masih_di_tps','ditetapkan','pemberitahuan'],true)) return 'warning'; return 'info'; }
function tpl_can(mixed $user, mixed $permission): bool { if ((string)tpl_get($user,'Role') === 'admin') return true; return in_array((string)$permission, (array)tpl_get($user,'Permissions',[]), true); }
function tpl_has_permission(mixed $permissions, mixed $permission): bool { return in_array((string)$permission,(array)$permissions,true); }
function tpl_applies_to(mixed $types, mixed $type): bool { return in_array((string)$type,array_map('trim',explode(',',(string)$types)),true); }
function tpl_parameter_group_label(mixed $code): string { return match((string)$code) {'bdn_category'=>'Kategori BDN','item_kind'=>'Jenis barang','goods_condition'=>'Kondisi barang','unit'=>'Satuan barang','allocation_purpose'=>'Peruntukan BMMN','origin_tps'=>'TPS asal','tpp'=>'Nama TPP','load_type'=>'Jenis muatan','exit_type'=>'Jenis pengeluaran','transfer_type'=>'Jenis serah terima',default=>(string)$code}; }
function tpl_change_section(mixed $section): string { return match((string)$section) {'inventory'=>'Data utama barang','event'=>'Dokumen/tahapan','process'=>'Data proses penyelesaian',default=>(string)$section}; }
function tpl_change_field(mixed $field): string {
    static $labels=['reference_no'=>'Nomor referensi','item_type'=>'Jenis inventory','origin_type'=>'Jenis inventory asal','bl_no'=>'Nomor BL','bl_date'=>'Tanggal BL','manifest_no'=>'Nomor manifest','manifest_date'=>'Tanggal manifest','manifest_position'=>'Pos manifest','determination_no'=>'Nomor penetapan atau dokumen dasar','determination_date'=>'Tanggal penetapan atau dokumen dasar','category'=>'Kategori BDN','entrusted_category'=>'Kategori barang titipan','source_office'=>'Kantor atau unit penitip','description'=>'Uraian barang','item_kind'=>'Jenis barang','quantity'=>'Jumlah barang','quantity_detail'=>'Detail jumlah barang','unit'=>'Satuan barang','goods_value'=>'Nilai barang','goods_condition'=>'Kondisi barang','location'=>'Lokasi atau blok','location_status'=>'Status lokasi','at_tpp'=>'Keberadaan di TPP','owner_name'=>'Nama pemilik','owner_address'=>'Alamat pemilik','origin_warehouse'=>'TPS asal','facility_id'=>'ID TPP','facility_name'=>'Nama TPP','load_type'=>'Jenis muatan','container_no'=>'Nomor kontainer','container_size'=>'Ukuran kontainer','estimated_volume_m3'=>'Perkiraan volume','physical_unit_id'=>'Identitas unit fisik','occupancy_primary'=>'Unit utama perhitungan kapasitas','pfpd_required'=>'Memerlukan penelitian PFPD','research_request_no'=>'Nomor request penelitian PFPD','research_request_date'=>'Tanggal request penelitian PFPD','hs_code'=>'HS Code','is_restricted'=>'Status lartas','restriction_rule'=>'Ketentuan lartas','allocation_purpose'=>'Peruntukan BMMN','exit_document_no'=>'Nomor dokumen pengeluaran','exit_document_date'=>'Tanggal dokumen pengeluaran','exit_type'=>'Jenis pengeluaran','exit_notes'=>'Catatan pengeluaran','label'=>'Nama tahapan','document_no'=>'Nomor dokumen','document_date'=>'Tanggal dokumen','notes'=>'Catatan tahapan','sale_value'=>'Nilai jual','htl_value'=>'Nilai HTL','destruction_cost'=>'Biaya pemusnahan','transfer_type'=>'Jenis serah terima'];
    return $labels[(string)$field] ?? (string)$field;
}
function tpl_change_value(mixed $field, mixed $value): string { if ($value === '' || $value === null) return '—'; if (in_array((string)$field,['goods_value','sale_value','htl_value','destruction_cost'],true) && is_numeric($value)) return tpl_rupiah($value); if (in_array((string)$field,['at_tpp','occupancy_primary','pfpd_required','is_restricted'],true)) return in_array(strtolower((string)$value),['1','true','ya'],true)?'Ya':'Tidak'; if (str_contains((string)$field,'date')) return tpl_date($value); return (string)$value; }
