<?php
declare(strict_types=1);

namespace Livira\Support;

final class Xlsx
{
    /** @return array<int,array<int,string>> */
    public static function read(string $file, int $maxRows = 1001): array
    {
        if (!is_file($file)) {
            throw new \RuntimeException('File Excel tidak ditemukan.');
        }
        $dir = self::tempDir('xlsx-read-');
        try {
            self::extract($file, $dir);
            $shared = [];
            $sharedFile = $dir.'/xl/sharedStrings.xml';
            if (is_file($sharedFile)) {
                $xml = (string) file_get_contents($sharedFile);
                if (preg_match_all('/<(?:[A-Za-z_][\w.-]*:)?si\b[^>]*>(.*?)<\/(?:[A-Za-z_][\w.-]*:)?si>/s', $xml, $matches)) {
                    foreach ($matches[1] as $si) {
                        preg_match_all('/<(?:[A-Za-z_][\w.-]*:)?t\b[^>]*>(.*?)<\/(?:[A-Za-z_][\w.-]*:)?t>/s', $si, $texts);
                        $shared[] = self::xmlText(implode('', $texts[1] ?? []));
                    }
                }
            }
            $sheet = $dir.'/xl/worksheets/sheet1.xml';
            if (!is_file($sheet)) {
                throw new \RuntimeException('Worksheet pertama tidak ditemukan.');
            }
            $xml = (string) file_get_contents($sheet);
            $rows = [];
            if (preg_match_all('/<(?:[A-Za-z_][\w.-]*:)?row\b[^>]*>(.*?)<\/(?:[A-Za-z_][\w.-]*:)?row>/s', $xml, $rowMatches)) {
                foreach ($rowMatches[1] as $rowXml) {
                    $row = [];
                    if (preg_match_all('/<(?:[A-Za-z_][\w.-]*:)?c\b([^>]*?)(?:\/>|>(.*?)<\/(?:[A-Za-z_][\w.-]*:)?c>)/s', $rowXml, $cells, PREG_SET_ORDER)) {
                        foreach ($cells as $cell) {
                            $attrs = $cell[1];
                            $body = (string) ($cell[2] ?? '');
                            preg_match('/\br="([A-Z]+)[0-9]+"/', $attrs, $rm);
                            $column = self::columnIndex($rm[1] ?? 'A');
                            $type = '';
                            if (preg_match('/\bt="([^"]+)"/', $attrs, $tm)) {
                                $type = $tm[1];
                            }
                            $value = '';
                            if ($type === 'inlineStr') {
                                if (preg_match_all('/<(?:[A-Za-z_][\w.-]*:)?t\b[^>]*>(.*?)<\/(?:[A-Za-z_][\w.-]*:)?t>/s', $body, $tmatches)) {
                                    $value = self::xmlText(implode('', $tmatches[1]));
                                }
                            } elseif (preg_match('/<(?:[A-Za-z_][\w.-]*:)?v\b[^>]*>(.*?)<\/(?:[A-Za-z_][\w.-]*:)?v>/s', $body, $vm)) {
                                $raw = self::xmlText($vm[1]);
                                $value = $type === 's' ? ($shared[(int) $raw] ?? '') : $raw;
                            }
                            $row[$column] = $value;
                        }
                    }
                    if ($row !== []) {
                        $last = max(array_keys($row));
                        $normalized = [];
                        for ($i = 0; $i <= $last; $i++) {
                            $normalized[] = (string) ($row[$i] ?? '');
                        }
                        $hasValue = false;
                        foreach ($normalized as $cellValue) {
                            if (trim($cellValue) !== '') {
                                $hasValue = true;
                                break;
                            }
                        }
                        if (!$hasValue) {
                            continue;
                        }
                        $rows[] = $normalized;
                        if (count($rows) >= $maxRows) {
                            break;
                        }
                    }
                }
            }
            return $rows;
        } finally {
            self::removeDir($dir);
        }
    }

    /** @param array<int,string> $headers @param array<int,array<int|string,mixed>> $rows */
    public static function write(array $headers, array $rows, string $sheetName = 'Laporan'): string
    {
        return self::writeSheets([[
            'name' => $sheetName,
            'headers' => $headers,
            'rows' => $rows,
        ]]);
    }

    /**
     * @param array<int,array{name:string,headers:array<int,string>,rows:array<int,array<int|string,mixed>>}> $sheets
     */
    public static function writeSheets(array $sheets): string
    {
        if ($sheets === []) {
            throw new \InvalidArgumentException('Minimal satu sheet diperlukan.');
        }

        $dir = self::tempDir('xlsx-write-');
        try {
            @mkdir($dir.'/_rels', 0775, true);
            @mkdir($dir.'/xl/_rels', 0775, true);
            @mkdir($dir.'/xl/worksheets', 0775, true);
            @mkdir($dir.'/docProps', 0775, true);

            $usedNames = [];
            $workbookSheets = '';
            $workbookRelationships = '';
            $contentOverrides = '';

            foreach (array_values($sheets) as $index => $definition) {
                $sheetId = $index + 1;
                $headers = array_values((array) ($definition['headers'] ?? []));
                $rows = array_values((array) ($definition['rows'] ?? []));
                $name = self::uniqueSheetName((string) ($definition['name'] ?? ('Sheet '.$sheetId)), $usedNames);
                $usedNames[] = $name;

                file_put_contents(
                    $dir.'/xl/worksheets/sheet'.$sheetId.'.xml',
                    self::worksheetXml($headers, $rows)
                );

                $workbookSheets .= '<sheet name="'.self::xml($name).'" sheetId="'.$sheetId.'" r:id="rId'.$sheetId.'"/>';
                $workbookRelationships .= '<Relationship Id="rId'.$sheetId.'" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet'.$sheetId.'.xml"/>';
                $contentOverrides .= '<Override PartName="/xl/worksheets/sheet'.$sheetId.'.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>';
            }

            $styleRelationshipId = count($sheets) + 1;
            $workbookRelationships .= '<Relationship Id="rId'.$styleRelationshipId.'" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>';

            file_put_contents($dir.'/[Content_Types].xml', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
                .'<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">'
                .'<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
                .'<Default Extension="xml" ContentType="application/xml"/>'
                .'<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>'
                .$contentOverrides
                .'<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>'
                .'<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>'
                .'<Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>'
                .'</Types>');

            file_put_contents($dir.'/_rels/.rels', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
                .'<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
                .'<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>'
                .'<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>'
                .'<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>'
                .'</Relationships>');

            file_put_contents($dir.'/xl/workbook.xml', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
                .'<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">'
                .'<sheets>'.$workbookSheets.'</sheets>'
                .'</workbook>');

            file_put_contents($dir.'/xl/_rels/workbook.xml.rels', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
                .'<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
                .$workbookRelationships
                .'</Relationships>');

            file_put_contents($dir.'/xl/styles.xml', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
                .'<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">'
                .'<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>'
                .'<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>'
                .'<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>'
                .'<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>'
                .'<cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/></cellXfs>'
                .'</styleSheet>');

            $date = gmdate('Y-m-d\TH:i:s\Z');
            file_put_contents($dir.'/docProps/core.xml', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
                .'<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">'
                .'<dc:creator>LIVIRA PHP</dc:creator><dcterms:created xsi:type="dcterms:W3CDTF">'.$date.'</dcterms:created>'
                .'</cp:coreProperties>');
            file_put_contents($dir.'/docProps/app.xml', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
                .'<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>LIVIRA PHP</Application></Properties>');

            $zip = tempnam(sys_get_temp_dir(), 'livira-xlsx-');
            if ($zip === false) {
                throw new \RuntimeException('Gagal membuat file sementara.');
            }
            @unlink($zip);
            $zip .= '.xlsx';
            self::archive($dir, $zip);
            $content = (string) file_get_contents($zip);
            @unlink($zip);
            return $content;
        } finally {
            self::removeDir($dir);
        }
    }

    /** @param array<int,string> $headers @param array<int,array<int|string,mixed>> $rows */
    public static function csv(array $headers, array $rows): string
    {
        $f = fopen('php://temp', 'r+');
        if ($f === false) {
            throw new \RuntimeException('Gagal membuat CSV.');
        }
        fwrite($f, "\xEF\xBB\xBF");
        fputcsv($f, $headers, ';');
        foreach ($rows as $row) {
            if (!array_is_list($row)) {
                $normalized = [];
                foreach ($headers as $header) {
                    $normalized[] = $row[$header] ?? '';
                }
                $row = $normalized;
            }
            fputcsv($f, $row, ';');
        }
        rewind($f);
        $out = (string) stream_get_contents($f);
        fclose($f);
        return $out;
    }

    /** @param array<int,string> $headers @param array<int,array<int|string,mixed>> $rows */
    private static function worksheetXml(array $headers, array $rows): string
    {
        $normalizedRows = array_map(static function (array $row) use ($headers): array {
            if (array_is_list($row)) {
                return array_values($row);
            }
            $normalized = [];
            foreach ($headers as $header) {
                $normalized[] = $row[$header] ?? '';
            }
            return $normalized;
        }, $rows);
        $all = array_merge([$headers], $normalizedRows);
        $xml = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            .'<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">'
            .'<sheetViews><sheetView workbookViewId="0"/></sheetViews>'
            .'<sheetFormatPr defaultRowHeight="15"/><sheetData>';

        foreach ($all as $rowIndex => $row) {
            $excelRow = $rowIndex + 1;
            $xml .= '<row r="'.$excelRow.'">';
            foreach (array_values($row) as $columnIndex => $value) {
                $reference = self::columnName($columnIndex).$excelRow;
                $style = $rowIndex === 0 ? ' s="1"' : '';
                if (is_int($value) || is_float($value)) {
                    $xml .= '<c r="'.$reference.'"'.$style.'><v>'.$value.'</v></c>';
                } else {
                    $xml .= '<c r="'.$reference.'" t="inlineStr"'.$style.'><is><t xml:space="preserve">'.self::xml((string) $value).'</t></is></c>';
                }
            }
            $xml .= '</row>';
        }

        $lastColumn = self::columnName(max(0, count($headers) - 1));
        $lastRow = max(1, count($all));
        $xml .= '</sheetData>';
        if ($headers !== []) {
            $xml .= '<autoFilter ref="A1:'.$lastColumn.$lastRow.'"/>';
        }
        $xml .= '<pageMargins left="0.7" right="0.7" top="0.75" bottom="0.75" header="0.3" footer="0.3"/>'
            .'</worksheet>';
        return $xml;
    }

    /** @param array<int,string> $used */
    private static function uniqueSheetName(string $name, array $used): string
    {
        $name = trim((string) preg_replace('~[\\\\/?*\[\]:]+~u', ' ', $name));
        $name = mb_substr($name === '' ? 'Sheet' : $name, 0, 31);
        $candidate = $name;
        $suffix = 2;
        while (in_array(mb_strtolower($candidate), array_map('mb_strtolower', $used), true)) {
            $tail = ' '.$suffix++;
            $candidate = mb_substr($name, 0, 31 - mb_strlen($tail)).$tail;
        }
        return $candidate;
    }

    private static function xml(string $value): string
    {
        return htmlspecialchars($value, ENT_XML1 | ENT_QUOTES, 'UTF-8');
    }

    private static function tempDir(string $prefix): string
    {
        $dir = sys_get_temp_dir().'/'.$prefix.bin2hex(random_bytes(8));
        if (!mkdir($dir, 0775, true) && !is_dir($dir)) {
            throw new \RuntimeException('Gagal membuat direktori sementara.');
        }
        return $dir;
    }

    private static function extract(string $file, string $dir): void
    {
        if (class_exists('ZipArchive')) {
            $zip = new \ZipArchive();
            if ($zip->open($file) !== true) {
                throw new \RuntimeException('File XLSX tidak dapat dibuka.');
            }
            $zip->extractTo($dir);
            $zip->close();
            return;
        }
        $command = 'unzip -qq '.escapeshellarg($file).' -d '.escapeshellarg($dir).' 2>&1';
        exec($command, $output, $code);
        if ($code !== 0) {
            throw new \RuntimeException('File XLSX tidak valid: '.implode(' ', $output));
        }
    }

    private static function archive(string $dir, string $file): void
    {
        if (class_exists('ZipArchive')) {
            $zip = new \ZipArchive();
            if ($zip->open($file, \ZipArchive::CREATE | \ZipArchive::OVERWRITE) !== true) {
                throw new \RuntimeException('Gagal membuat XLSX.');
            }
            $iterator = new \RecursiveIteratorIterator(new \RecursiveDirectoryIterator($dir, \FilesystemIterator::SKIP_DOTS));
            foreach ($iterator as $entry) {
                $zip->addFile($entry->getPathname(), substr($entry->getPathname(), strlen($dir) + 1));
            }
            $zip->close();
            return;
        }
        $cwd = getcwd();
        chdir($dir);
        exec('zip -qr '.escapeshellarg($file).' . 2>&1', $output, $code);
        chdir($cwd ?: '/');
        if ($code !== 0) {
            throw new \RuntimeException('Gagal membuat XLSX: '.implode(' ', $output));
        }
    }

    private static function removeDir(string $dir): void
    {
        if (!is_dir($dir)) {
            return;
        }
        $iterator = new \RecursiveIteratorIterator(
            new \RecursiveDirectoryIterator($dir, \FilesystemIterator::SKIP_DOTS),
            \RecursiveIteratorIterator::CHILD_FIRST
        );
        foreach ($iterator as $entry) {
            $entry->isDir() ? @rmdir($entry->getPathname()) : @unlink($entry->getPathname());
        }
        @rmdir($dir);
    }

    private static function xmlText(string $value): string
    {
        return html_entity_decode(strip_tags($value), ENT_QUOTES | ENT_XML1, 'UTF-8');
    }

    private static function columnIndex(string $letters): int
    {
        $number = 0;
        foreach (str_split($letters) as $letter) {
            $number = $number * 26 + (ord($letter) - 64);
        }
        return max(0, $number - 1);
    }

    private static function columnName(int $index): string
    {
        $index++;
        $name = '';
        while ($index > 0) {
            $index--;
            $name = chr(65 + $index % 26).$name;
            $index = intdiv($index, 26);
        }
        return $name;
    }
}
