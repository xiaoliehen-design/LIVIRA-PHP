<?php
declare(strict_types=1);
namespace Livira\Supabase;

final class SupabaseClient
{
    public function __construct(private readonly string $projectUrl, private readonly string $key, private readonly int $timeout=25) {}
    public function rest(string $method,string $resource,array $query=[],mixed $body=null,array $headers=[]):mixed
    {
        $url=rtrim($this->projectUrl,'/').'/rest/v1/'.ltrim($resource,'/');
        if($query)$url.='?'.$this->encodeQuery($query);
        $h=['apikey: '.$this->key,'Authorization: Bearer '.$this->key,'Accept: application/json'];
        if($body!==null){$h[]='Content-Type: application/json';$payload=json_encode($body,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES|JSON_THROW_ON_ERROR);}else{$payload='';}
        if(in_array(strtoupper($method),['POST','PATCH','DELETE'],true))$h[]='Prefer: return=representation';
        foreach($headers as $k=>$v)$h[]=$k.': '.$v;
        [$status,$resp,$respHeaders]=$this->http($method,$url,$h,$payload);
        if($status<200||$status>=300)throw new ApiException('Supabase '.strtoupper($method).' '.$resource.' gagal (HTTP '.$status.'): '.trim($resp),$status,$resp);
        if(trim($resp)==='')return null;
        $decoded=json_decode($resp,true);if(json_last_error()!==JSON_ERROR_NONE)throw new ApiException('Respons Supabase tidak valid: '.json_last_error_msg(),$status,$resp);
        return $decoded;
    }
    public function restCount(string $resource,array $query=[]):int
    {
        $query['select']='id';$query['limit']='1';$url=rtrim($this->projectUrl,'/').'/rest/v1/'.ltrim($resource,'/').'?'.$this->encodeQuery($query);
        [$status,$resp,$headers]=$this->http('GET',$url,['apikey: '.$this->key,'Authorization: Bearer '.$this->key,'Prefer: count=exact','Range: 0-0','Accept: application/json'],'');
        if($status<200||$status>=300)throw new ApiException('Supabase count gagal (HTTP '.$status.'): '.$resp,$status,$resp);
        $range=$headers['content-range']??'';if(!str_contains($range,'/'))return 0;return (int)substr($range,strrpos($range,'/')+1);
    }
    public function auth(string $method,string $path,mixed $body=null,?string $bearer=null):mixed
    {
        $url=rtrim($this->projectUrl,'/').'/auth/v1/'.ltrim($path,'/');$payload=$body===null?'':json_encode($body,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES|JSON_THROW_ON_ERROR);
        $h=['apikey: '.$this->key,'Authorization: Bearer '.($bearer?:$this->key),'Accept: application/json'];if($body!==null)$h[]='Content-Type: application/json';
        [$status,$resp]=$this->http($method,$url,$h,$payload);if($status<200||$status>=300){$e=json_decode($resp,true);$msg=$e['msg']??$e['error_description']??$e['message']??trim($resp);throw new ApiException('Supabase Auth: '.$msg,$status,$resp);}return trim($resp)===''?null:json_decode($resp,true,512,JSON_THROW_ON_ERROR);
    }
    public function storage(string $method,string $bucket,string $objectPath,string $mime='',string $body=''):string
    {
        $segments=array_map('rawurlencode',explode('/',$objectPath));$url=rtrim($this->projectUrl,'/').'/storage/v1/object/'.rawurlencode($bucket).'/'.implode('/',$segments);
        $h=['apikey: '.$this->key,'Authorization: Bearer '.$this->key];if($mime!=='')$h[]='Content-Type: '.$mime;if(strtoupper($method)==='POST')$h[]='x-upsert: false';
        [$status,$resp]=$this->http($method,$url,$h,$body);if($status<200||$status>=300)throw new ApiException('Supabase Storage gagal (HTTP '.$status.'): '.trim($resp),$status,$resp);return $resp;
    }
    private function http(string $method,string $url,array $headers,string $body):array
    {
        $opts=['http'=>['method'=>strtoupper($method),'header'=>implode("\r\n",$headers),'content'=>$body,'ignore_errors'=>true,'timeout'=>$this->timeout,'protocol_version'=>1.1]];
        $context=stream_context_create($opts);$resp=@file_get_contents($url,false,$context);$resp=$resp===false?'':$resp;$raw=$http_response_header??[];$status=0;$out=[];
        foreach($raw as $i=>$line){if($i===0&&preg_match('/\s(\d{3})\s/',$line,$m))$status=(int)$m[1];elseif(str_contains($line,':')){[$k,$v]=explode(':',$line,2);$out[strtolower(trim($k))]=trim($v);}}
        if($status===0)throw new ApiException('Tidak dapat terhubung ke Supabase. Periksa URL, DNS, dan koneksi server.',503,$resp);
        return [$status,$resp,$out];
    }
    private function encodeQuery(array $query):string
    {
        $pairs=[];foreach($query as $k=>$v){foreach(is_array($v)?$v:[$v] as $item){if($item===null)continue;$pairs[]=rawurlencode((string)$k).'='.rawurlencode((string)$item);}}return implode('&',$pairs);
    }
}
