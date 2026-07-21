<?php
declare(strict_types=1);
namespace Livira\Http;

final class Response
{
    public function __construct(public string $body='', public int $status=200, public array $headers=[]) {}
    public static function html(string $body,int $status=200): self { return new self($body,$status,['Content-Type'=>'text/html; charset=utf-8']); }
    public static function json(mixed $data,int $status=200): self { return new self((string)json_encode($data,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),$status,['Content-Type'=>'application/json; charset=utf-8','Cache-Control'=>'no-store']); }
    public static function redirect(string $url,int $status=303): self { return new self('',$status,['Location'=>$url]); }
    public static function noContent(): self { return new self('',204); }
    public static function file(string $content,string $type,string $name): self { return new self($content,200,['Content-Type'=>$type,'Content-Disposition'=>'attachment; filename="'.str_replace('"','',$name).'"','Content-Length'=>(string)strlen($content),'X-Content-Type-Options'=>'nosniff']); }
    public function withCookie(string $cookie): self { $this->headers['Set-Cookie']=$cookie; return $this; }
    public function send(): never
    {
        http_response_code($this->status);
        foreach ($this->headers as $k=>$v) {
            if (is_array($v)) foreach($v as $line) header($k.': '.$line,false); else header($k.': '.$v,true);
        }
        echo $this->body; exit;
    }
}
