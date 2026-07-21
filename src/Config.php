<?php
declare(strict_types=1);
namespace Livira;

final class Config
{
    public function __construct(
        public readonly string $appName,
        public readonly string $appEnv,
        public readonly string $publicBaseUrl,
        public readonly string $sessionSecret,
        public readonly string $adminUsername,
        public readonly string $adminPassword,
        public readonly string $supabaseUrl,
        public readonly string $supabaseAnonKey,
        public readonly string $supabaseServiceKey,
        public readonly string $storageBucket,
        public readonly bool $demoMode,
        public readonly int $idleTimeoutSeconds = 1800,
    ) {}

    public static function load(string $basePath): self
    {
        self::loadDotEnv($basePath.'/.env');
        $url = rtrim(self::env('SUPABASE_URL'),'/');
        $anon = self::env('SUPABASE_ANON_KEY');
        $service = self::env('SUPABASE_SERVICE_ROLE_KEY');
        $configured = $url !== '' && $anon !== '' && $service !== '';
        $env = self::env('APP_ENV','development');
        $demo = filter_var(self::env('DEMO_MODE',$configured?'false':'true'), FILTER_VALIDATE_BOOL);
        $adminUser = trim(self::env('ADMIN_USERNAME'));
        $adminPassword = self::env('ADMIN_PASSWORD');
        if ($env !== 'production' && $demo && $adminUser === '' && $adminPassword === '') {
            $adminUser='admin'; $adminPassword='admin-demo-only';
        }
        $secret = self::env('SESSION_SECRET','change-this-session-secret-before-production');
        if ($env === 'production' && strlen($secret) < 32) throw new \RuntimeException('SESSION_SECRET production minimal 32 karakter.');
        if ($env === 'production' && $demo) throw new \RuntimeException('DEMO_MODE harus false pada production.');
        if ($env === 'production' && !$configured) throw new \RuntimeException('SUPABASE_URL, SUPABASE_ANON_KEY, dan SUPABASE_SERVICE_ROLE_KEY wajib diisi pada production.');
        if (($adminUser === '') xor ($adminPassword === '')) throw new \RuntimeException('ADMIN_USERNAME dan ADMIN_PASSWORD harus diisi keduanya atau dikosongkan keduanya.');
        if ($env === 'production' && $adminPassword !== '' && strlen($adminPassword) < 16) throw new \RuntimeException('ADMIN_PASSWORD production minimal 16 karakter.');
        return new self(
            self::env('APP_NAME','LIVIRA'),$env,rtrim(self::env('PUBLIC_BASE_URL'),'/'),$secret,
            $adminUser,$adminPassword,$url,$anon,$service,self::env('SUPABASE_STORAGE_BUCKET','livira-documents'),$demo,
            max(300,(int)self::env('IDLE_TIMEOUT_SECONDS','1800')),
        );
    }

    public function production(): bool { return strtolower($this->appEnv)==='production'; }
    public function supabaseConfigured(): bool { return $this->supabaseUrl!=='' && $this->supabaseAnonKey!=='' && $this->supabaseServiceKey!==''; }

    private static function env(string $key,string $fallback=''): string
    {
        $v=$_ENV[$key]??$_SERVER[$key]??getenv($key);
        return $v===false||$v===null||$v===''?$fallback:(string)$v;
    }
    private static function loadDotEnv(string $file): void
    {
        if (!is_file($file)) return;
        foreach (file($file, FILE_IGNORE_NEW_LINES|FILE_SKIP_EMPTY_LINES) ?: [] as $line) {
            $line=trim($line); if ($line===''||str_starts_with($line,'#')||!str_contains($line,'=')) continue;
            [$k,$v]=explode('=',$line,2); $k=trim($k); $v=trim($v);
            if (($v[0]??'')==='"' && str_ends_with($v,'"')) $v=stripcslashes(substr($v,1,-1));
            if (($v[0]??'')==="'" && str_ends_with($v,"'")) $v=substr($v,1,-1);
            if (getenv($k)===false) { $_ENV[$k]=$v; putenv($k.'='.$v); }
        }
    }
}
