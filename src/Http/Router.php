<?php
declare(strict_types=1);
namespace Livira\Http;

final class Router
{
    /** @var array<int,array{0:string,1:string,2:callable,3:array}> */
    private array $routes=[];

    public function add(string $method,string $pattern,callable $handler,array $middleware=[]): void
    {
        $this->routes[]=[strtoupper($method),$pattern,$handler,$middleware];
    }
    public function get(string $pattern,callable $handler,array $middleware=[]): void { $this->add('GET',$pattern,$handler,$middleware); }
    public function post(string $pattern,callable $handler,array $middleware=[]): void { $this->add('POST',$pattern,$handler,$middleware); }

    public function dispatch(Request $request): Response
    {
        $requestMethod=$request->method==='HEAD'?'GET':$request->method;
        foreach($this->routes as [$method,$pattern,$handler,$middleware]) {
            if ($method!==$requestMethod) continue;
            [$regex,$keys]=$this->compile($pattern);
            if(!preg_match($regex,$request->path,$matches)) continue;
            array_shift($matches);
            $params=[];
            foreach($keys as $index=>$key)$params[$key]=rawurldecode((string)($matches[$index]??''));
            $request->attributes['route']=$params;
            $next=static fn(Request $r):Response=>$handler($r);
            foreach(array_reverse($middleware) as $mw){$previous=$next;$next=static fn(Request $r):Response=>$mw($r,$previous);}
            return $next($request);
        }
        return Response::html('<!doctype html><html lang="id"><meta charset="utf-8"><title>404</title><body><h1>404</h1><p>Halaman tidak ditemukan.</p></body></html>',404);
    }

    /** @return array{0:string,1:array<int,string>} */
    private function compile(string $pattern): array
    {
        $keys=[];$cursor=0;$regex='';
        if(preg_match_all('/\{([A-Za-z_][A-Za-z0-9_]*)\}/',$pattern,$matches,PREG_OFFSET_CAPTURE)){
            foreach($matches[0] as $i=>$match){
                [$token,$offset]=$match;
                $regex.=preg_quote(substr($pattern,$cursor,$offset-$cursor),'#').'([^/]+)';
                $keys[]=$matches[1][$i][0];
                $cursor=$offset+strlen($token);
            }
        }
        $regex.=preg_quote(substr($pattern,$cursor),'#');
        return ['#^'.$regex.'$#D',$keys];
    }
}
