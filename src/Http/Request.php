<?php
declare(strict_types=1);
namespace Livira\Http;

final class Request
{
    public readonly string $method;
    public readonly string $path;
    public readonly array $query;
    public readonly array $form;
    public readonly array $files;
    public readonly array $headers;
    public readonly string $body;
    public array $attributes=[];

    public function __construct(?string $method=null, ?string $uri=null, ?array $query=null, ?array $form=null, ?array $files=null, ?array $headers=null, ?string $body=null)
    {
        $this->method=strtoupper($method??($_SERVER['REQUEST_METHOD']??'GET'));
        $raw=$uri??($_SERVER['REQUEST_URI']??'/');
        $this->path=rawurldecode(parse_url($raw,PHP_URL_PATH)?:'/');
        $this->query=$query??$_GET;
        $this->form=$form??$_POST;
        $this->files=$files??$_FILES;
        $this->headers=$headers??$this->readHeaders();
        $this->body=$body??(string)file_get_contents('php://input');
    }
    public function input(string $key,mixed $default=''): mixed { return $this->form[$key]??$this->query[$key]??$default; }
    public function query(string $key,mixed $default=''): mixed { return $this->query[$key]??$default; }
    public function header(string $key,string $default=''): string { return (string)($this->headers[strtolower($key)]??$default); }
    public function acceptsJson(): bool { return str_contains(strtolower($this->header('accept')),'application/json')||str_starts_with($this->path,'/api/'); }
    public function ip(): string { $raw=$this->header('x-forwarded-for',$_SERVER['REMOTE_ADDR']??''); return trim(explode(',',$raw)[0]); }
    public function userAgent(): string { return substr($this->header('user-agent'),0,500); }
    public function route(string $key,mixed $default=''): mixed { return $this->attributes['route'][$key]??$default; }
    public function json(): array { $v=json_decode($this->body,true); return is_array($v)?$v:[]; }
    private function readHeaders(): array { $h=[]; foreach ($_SERVER as $k=>$v) if (str_starts_with($k,'HTTP_')) $h[strtolower(str_replace('_','-',substr($k,5)))]=(string)$v; if (isset($_SERVER['CONTENT_TYPE']))$h['content-type']=$_SERVER['CONTENT_TYPE']; return $h; }
}
