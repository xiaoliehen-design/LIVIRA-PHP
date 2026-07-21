<?php
declare(strict_types=1);
namespace Livira\Supabase;

use Livira\Config;
use Livira\Security\SessionManager;

final class AuthClient
{
    private SupabaseClient $anon;
    public function __construct(private readonly Config $config, private readonly SessionManager $sessions){$this->anon=new SupabaseClient($config->supabaseUrl,$config->supabaseAnonKey);}
    public function login(string $identity,string $password):array
    {
        $identity=trim($identity);
        if($this->config->adminUsername!==''&&hash_equals($this->config->adminUsername,$identity)&&hash_equals($this->config->adminPassword,$password))return $this->sessions->adminSession($identity);
        if(!$this->config->supabaseConfigured())throw new ApiException('Login Supabase belum dikonfigurasi.',503);
        $r=$this->anon->auth('POST','token?grant_type=password',['email'=>strtolower($identity),'password'=>$password]);$u=$r['user']??[];if(($u['id']??'')==='')throw new ApiException('Kredensial tidak valid.',401);
        $name=trim((string)($u['user_metadata']['name']??$u['email']??$identity));
        return $this->sessions->create(['Subject'=>'user:'.$u['id'],'Email'=>$u['email']??$identity,'DisplayName'=>$name,'Role'=>'user']);
    }
    public function signup(string $name,string $email,string $password):array{$r=$this->anon->auth('POST','signup',['email'=>strtolower(trim($email)),'password'=>$password,'data'=>['name'=>trim($name)]]);$u=$r['user']??$r;if(($u['id']??'')==='')throw new ApiException('Pendaftaran tidak dapat dibuat atau email sudah digunakan.',400);return ['UserID'=>$u['id'],'Email'=>$u['email']??$email];}
    public function verifySignup(string $email,string $token):array{$r=$this->anon->auth('POST','verify',['email'=>strtolower(trim($email)),'token'=>trim($token),'type'=>'email']);$u=$r['user']??[];if(($u['id']??'')==='')throw new ApiException('OTP tidak valid atau kedaluwarsa.',400);return ['UserID'=>$u['id'],'Email'=>$u['email']??$email,'Name'=>$u['user_metadata']['name']??$email];}
    public function resendSignup(string $email):void{$this->anon->auth('POST','resend',['email'=>strtolower(trim($email)),'type'=>'signup']);}
    public function requestPasswordReset(string $email):void{$this->anon->auth('POST','recover',['email'=>strtolower(trim($email))]);}
    public function resetPassword(string $email,string $token,string $password):void{$r=$this->anon->auth('POST','verify',['email'=>strtolower(trim($email)),'token'=>trim($token),'type'=>'recovery']);$access=$r['access_token']??'';if($access==='')throw new ApiException('OTP pemulihan tidak menghasilkan sesi valid.',400);$this->anon->auth('PUT','user',['password'=>$password],$access);}
}
