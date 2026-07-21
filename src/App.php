<?php
declare(strict_types=1);
namespace Livira;

use Livira\Domain\Domain;
use Livira\Http\Request;
use Livira\Http\Response;
use Livira\Http\Router;
use Livira\Security\Captcha;
use Livira\Security\RateLimiter;
use Livira\Security\SessionManager;
use Livira\Supabase\ApiException;
use Livira\Supabase\AuthClient;
use Livira\Supabase\DemoStore;
use Livira\Supabase\Store;
use Livira\Supabase\SupabaseClient;
use Livira\Support\View;
use Livira\Support\Xlsx;

final class App
{
    private readonly Router $router;
    private readonly SessionManager $sessions;
    private readonly Captcha $captcha;
    private readonly RateLimiter $limiter;
    private readonly View $view;
    private readonly AuthClient $auth;
    private readonly object $store;

    public function __construct(private readonly string $basePath,private readonly Config $config)
    {
        $this->sessions=new SessionManager($config);
        $this->captcha=new Captcha($config->sessionSecret,$basePath.'/storage/cache');
        $this->limiter=new RateLimiter($basePath.'/storage/cache/rate-limits.json');
        $this->view=new View($basePath.'/resources/views');
        $this->auth=new AuthClient($config,$this->sessions);
        $this->store=$config->demoMode
            ? new DemoStore($basePath.'/storage/demo-data.json',$basePath.'/storage/demo-documents')
            : new Store($config,new SupabaseClient($config->supabaseUrl,$config->supabaseServiceKey));
        $this->router=new Router();$this->registerRoutes();
    }

    public function handle(Request $request):Response
    {
        try{$response=$this->router->dispatch($request);}catch(ApiException $e){$response=$this->errorResponse($request,$e->getMessage(),$this->httpStatus($e));}catch(\Throwable $e){$this->logException($e,$request);$response=$this->errorResponse($request,$this->config->production()?'Terjadi kesalahan internal. Silakan coba kembali.':$e->getMessage(),500);}
        $response->headers+=['X-Content-Type-Options'=>'nosniff','X-Frame-Options'=>'DENY','Referrer-Policy'=>'same-origin','Permissions-Policy'=>'camera=(), microphone=(), geolocation=()','Cross-Origin-Opener-Policy'=>'same-origin','Cache-Control'=>$response->headers['Cache-Control']??'no-store'];
        if(str_starts_with((string)($response->headers['Content-Type']??''),'text/html'))$response->headers['Content-Security-Policy']="default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; font-src 'self'; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'self'";
        $this->auditRequest($request,$response);
        return$response;
    }

    private function registerRoutes():void
    {
        $p=fn(Request $r,callable $next)=>$this->protected($r,$next);
        $csrf=fn(Request $r,callable $next)=>$this->csrf($r,$next);
        $perm=fn(string $permission)=>fn(Request $r,callable $next)=>$this->permission($r,$next,$permission);
        $any=fn(array $permissions)=>fn(Request $r,callable $next)=>$this->anyPermission($r,$next,$permissions);
        $admin=fn(Request $r,callable $next)=>$this->adminOnly($r,$next);

        $this->router->get('/healthz',fn()=>Response::json(['status'=>'ok','app'=>'LIVIRA PHP','time'=>gmdate('c')]));
        $this->router->get('/login',fn(Request $r)=>$this->loginPage($r));
        $this->router->post('/login',fn(Request $r)=>$this->login($r));
        $this->router->get('/captcha.png',fn(Request $r)=>new Response($this->captcha->image((string)$r->query('token')),200,['Content-Type'=>'image/svg+xml; charset=utf-8','Cache-Control'=>'no-store']));
        $this->router->get('/captcha/new',fn()=> $this->newCaptcha());
        $this->router->get('/signup',fn(Request $r)=>$this->authPage($r,['SignupPage'=>true,'Title'=>'Pendaftaran akun']));
        $this->router->post('/signup',fn(Request $r)=>$this->signup($r));
        $this->router->get('/signup/verify',fn(Request $r)=>$this->authPage($r,['OTPPage'=>true,'VerifyEmail'=>(string)$r->query('email'),'Title'=>'Verifikasi OTP']));
        $this->router->post('/signup/verify',fn(Request $r)=>$this->verifySignup($r));
        $this->router->post('/signup/resend',fn(Request $r)=>$this->resendSignup($r));
        $this->router->get('/forgot-password',fn(Request $r)=>$this->authPage($r,['ForgotPasswordPage'=>true,'VerifyEmail'=>(string)$r->query('email'),'Title'=>'Lupa password']));
        $this->router->post('/forgot-password',fn(Request $r)=>$this->requestPasswordReset($r));
        $this->router->get('/forgot-password/verify',fn(Request $r)=>$this->authPage($r,['ResetPasswordPage'=>true,'VerifyEmail'=>(string)$r->query('email'),'Title'=>'Buat password baru']));
        $this->router->post('/forgot-password/verify',fn(Request $r)=>$this->resetPassword($r));

        $this->router->post('/logout',fn(Request $r)=>$this->logout($r),[$p,$csrf]);
        $this->router->post('/session/activity',fn(Request $r)=>Response::json(['ok'=>true,'expires_in'=>$this->config->idleTimeoutSeconds]),[$p,$csrf]);
        $this->router->post('/session/idle-logout',fn(Request $r)=>$this->logout($r),[$p,$csrf]);
        $this->router->get('/',fn(Request $r)=>$this->dashboard($r),[$p,$perm('dashboard.view')]);
        $this->router->post('/admin/facilities/{id}/capacity',fn(Request $r)=>$this->capacity($r),[$p,$csrf,$perm('dashboard.capacity.manage')]);
        $this->router->get('/inventory',fn(Request $r)=>$this->inventory($r),[$p,$perm('inventory.view')]);
        $this->router->post('/inventory',fn(Request $r)=>$this->createInventory($r),[$p,$csrf,$any($this->inventoryManagementPermissions())]);
        $this->router->post('/inventory/import',fn(Request $r)=>$this->importInventory($r),[$p,$csrf,$any($this->inventoryManagementPermissions())]);
        $this->router->post('/inventory/{id}/event',fn(Request $r)=>$this->inventoryEvent($r,false),[$p,$csrf,$any($this->inventoryManagementPermissions())]);
        $this->router->post('/inventory/bulk-event',fn(Request $r)=>$this->inventoryEvent($r,true),[$p,$csrf,$any($this->inventoryManagementPermissions())]);
        $this->router->post('/admin/inventory/{id}/delete',fn(Request $r)=>$this->deleteInventory($r),[$p,$csrf,$admin]);
        $this->router->get('/proses/{type}',fn(Request $r)=>$this->processPage($r),[$p]);
        $this->router->post('/proses/{type}/bulk-action',fn(Request $r)=>$this->processAction($r),[$p,$csrf]);
        $this->router->get('/rekonsiliasi',fn(Request $r)=>$this->reconciliation($r),[$p,$perm('reconciliation.view')]);
        $this->router->post('/rekonsiliasi',fn(Request $r)=>$this->reconcile($r),[$p,$csrf,$perm('reconciliation.manage')]);
        $this->router->get('/pelaporan',fn(Request $r)=>$this->reports($r),[$p,$perm('reports.view')]);
        $this->router->get('/pelaporan.csv',fn(Request $r)=>$this->exportReport($r,'csv'),[$p,$perm('reports.view')]);
        $this->router->get('/pelaporan.xlsx',fn(Request $r)=>$this->exportReport($r,'xlsx'),[$p,$perm('reports.view')]);
        $this->router->get('/pelaporan.xls',fn(Request $r)=>$this->exportReport($r,'xls'),[$p,$perm('reports.view')]);
        $this->router->get('/pelaporan/performa.xlsx',fn(Request $r)=>$this->exportPerformance($r),[$p,$perm('reports.view')]);
        $this->router->get('/pencarian',fn(Request $r)=>$this->searchPage($r),[$p,$perm('search.view')]);
        $viewAny=['inventory.view','auction.view','destruction.view','grant.view','search.view'];
        $this->router->get('/api/inventory/search',fn(Request $r)=>Response::json(['items'=>$this->store->listInventory(['query'=>(string)$r->query('q'),'include_inactive'=>true,'allowed_types'=>Domain::allowedTypes($this->session($r)),'limit'=>50])]),[$p,$any($viewAny)]);
        $this->router->get('/api/inventory/{id}',fn(Request $r)=>$this->inventoryDetail($r),[$p,$any($viewAny)]);
        $this->router->get('/api/inventory/{id}/timeline',fn(Request $r)=>$this->inventoryTimeline($r),[$p,$any($viewAny)]);
        $this->router->get('/api/proses/{id}/timeline',fn(Request $r)=>$this->processTimeline($r),[$p,$any(['auction.view','destruction.view','grant.view'])]);
        $this->router->get('/documents/{id}/download',fn(Request $r)=>$this->downloadDocument($r),[$p,$any(array_merge($viewAny,['reconciliation.view']))]);
        $this->router->get('/admin/pendaftaran',fn(Request $r)=>$this->adminUsers($r),[$p,$perm('admin.users')]);
        $this->router->post('/admin/pendaftaran/{id}/approve',fn(Request $r)=>$this->approveUser($r),[$p,$csrf,$perm('admin.users')]);
        $this->router->post('/admin/pendaftaran/{id}/reject',fn(Request $r)=>$this->rejectUser($r),[$p,$csrf,$perm('admin.users')]);
        $this->router->post('/admin/pendaftaran/{id}/role',fn(Request $r)=>$this->updateUserRole($r),[$p,$csrf,$perm('admin.users')]);
        $this->router->post('/admin/pendaftaran/{id}/delete',fn(Request $r)=>$this->deleteUser($r),[$p,$csrf,$perm('admin.users')]);
        $this->router->get('/admin/roles',fn(Request $r)=>$this->adminRoles($r),[$p,$perm('admin.roles')]);
        $this->router->post('/admin/roles',fn(Request $r)=>$this->createRole($r),[$p,$csrf,$perm('admin.roles')]);
        $this->router->post('/admin/roles/{id}/update',fn(Request $r)=>$this->updateRole($r),[$p,$csrf,$perm('admin.roles')]);
        $this->router->post('/admin/roles/{id}/status',fn(Request $r)=>$this->roleStatus($r),[$p,$csrf,$perm('admin.roles')]);
        $this->router->post('/admin/roles/{id}/delete',fn(Request $r)=>$this->deleteRole($r),[$p,$csrf,$perm('admin.roles')]);
        $this->router->get('/admin/parameters',fn(Request $r)=>$this->adminParameters($r),[$p,$perm('admin.parameters')]);
        $this->router->post('/admin/parameters',fn(Request $r)=>$this->createParameter($r),[$p,$csrf,$perm('admin.parameters')]);
        $this->router->post('/admin/parameters/{id}/update',fn(Request $r)=>$this->updateParameter($r),[$p,$csrf,$perm('admin.parameters')]);
        $this->router->post('/admin/parameters/{id}/status',fn(Request $r)=>$this->parameterStatus($r),[$p,$csrf,$perm('admin.parameters')]);
    }

    private function loginPage(Request $r):Response
    {
        if($this->sessions->read())return Response::redirect('/');
        [$token]=$this->captcha->challenge();return$this->authPage($r,['CaptchaToken'=>$token,'Title'=>'Masuk ke LIVIRA']);
    }
    private function authPage(Request $r,array $extra=[]):Response{return Response::html($this->view->render('auth',array_merge(['AuthPage'=>true,'DemoMode'=>$this->config->demoMode,'Success'=>(string)$r->query('success'),'Error'=>(string)$r->query('error')],$extra)));}
    private function newCaptcha():Response{[$token]=$this->captcha->challenge();return Response::json(['token'=>$token,'image_url'=>'/captcha.png?token='.rawurlencode($token),'expires_in'=>300]);}
    private function login(Request $r):Response
    {
        $key='login:'.$r->ip();if(!$this->limiter->allow($key,10,900))return$this->loginFailure($r,'Terlalu banyak percobaan login. Coba kembali beberapa saat lagi.',429);
        if(!$this->captcha->verify((string)$r->input('captcha_token'),(string)$r->input('captcha_answer')))return$this->loginFailure($r,'Kode CAPTCHA tidak sesuai.',422);
        $identity=trim((string)$r->input('identity'));$session=$this->auth->login($identity,(string)$r->input('password'));
        if(($session['Role']??'')!=='admin'){
            $authId=str_replace('user:','',(string)$session['Subject']);$user=$this->store->userByAuthId($authId);
            if(($user['approval_status']??'')!=='approved')throw new ApiException(($user['approval_status']??'')==='rejected'?'Pendaftaran ditolak: '.($user['rejection_reason']??''): 'Akun menunggu persetujuan administrator.',403);
            $session=array_replace($session,['DisplayName'=>$user['name']??$session['DisplayName'],'Email'=>$user['email']??$session['Email'],'RoleID'=>$user['role_id']??'','RoleName'=>$user['role_name']??'Pengguna','Permissions'=>Domain::normalizePermissions((array)($user['permissions']??[]))]);
        }
        $this->limiter->reset($key);return Response::redirect('/')->withCookie($this->sessions->cookie($session));
    }
    private function loginFailure(Request $r,string $message,int $status):Response{[$token]=$this->captcha->challenge();return Response::html($this->view->render('auth',['AuthPage'=>true,'Title'=>'Masuk ke LIVIRA','CaptchaToken'=>$token,'Error'=>$message,'DemoMode'=>$this->config->demoMode]),$status);}
    private function signup(Request $r):Response
    {
        $name=trim((string)$r->input('name'));$email=strtolower(trim((string)$r->input('email')));$password=(string)$r->input('password');if($name===''||!filter_var($email,FILTER_VALIDATE_EMAIL)||strlen($password)<8)throw new ApiException('Nama, email valid, dan password minimal 8 karakter wajib diisi.',422);
        $result=$this->auth->signup($name,$email,$password);$this->store->createUserApplication((string)$result['UserID'],$name,$email);return Response::redirect('/signup/verify?email='.rawurlencode($email).'&success='.rawurlencode('OTP 6 digit telah dikirim ke email.'));
    }
    private function verifySignup(Request $r):Response{$email=strtolower(trim((string)$r->input('email')));$result=$this->auth->verifySignup($email,(string)$r->input('token'));$this->store->markUserEmailVerified((string)$result['UserID'],$email);return Response::redirect('/login?success='.rawurlencode('Email terverifikasi. Pendaftaran menunggu persetujuan administrator.'));}
    private function resendSignup(Request $r):Response{$email=strtolower(trim((string)$r->input('email')));$this->auth->resendSignup($email);return Response::redirect('/signup/verify?email='.rawurlencode($email).'&success='.rawurlencode('OTP baru telah dikirim.'));}
    private function requestPasswordReset(Request $r):Response{$email=strtolower(trim((string)$r->input('email')));if(!filter_var($email,FILTER_VALIDATE_EMAIL))throw new ApiException('Email tidak valid.',422);$this->auth->requestPasswordReset($email);return Response::redirect('/forgot-password/verify?email='.rawurlencode($email).'&success='.rawurlencode('OTP reset password telah dikirim.'));}
    private function resetPassword(Request $r):Response{$password=(string)$r->input('password');if(strlen($password)<8||$password!==(string)$r->input('password_confirmation'))throw new ApiException('Password minimal 8 karakter dan konfirmasi harus sama.',422);$this->auth->resetPassword((string)$r->input('email'),(string)$r->input('token'),$password);return Response::redirect('/login?success='.rawurlencode('Password berhasil diubah. Silakan login.'));}
    private function logout(Request $r):Response{return Response::redirect('/login')->withCookie($this->sessions->clearCookie());}

    private function dashboard(Request $r):Response
    {
        $session=$this->session($r);$scope=(string)$r->query('inventory_scope','all_office');$facilityId=(string)$r->query('tpp');$facilities=$this->store->facilities();$global=$this->store->dashboard();
        $filter=['allowed_types'=>Domain::allowedTypes($session),'limit'=>50000];$label='Seluruh cakupan kantor Tanjung Priok';if($scope==='still_tps'){$filter['location_scope']='tps';$label='Barang yang masih berada di TPS';}elseif($scope==='all_tpp'){$filter['location_scope']='tpp';$label='Seluruh barang yang berada di TPP';}elseif($scope!==''&&$scope!=='all_office'){$filter['facility_id']=$scope;$filter['location_scope']='tpp';foreach($facilities as $f)if($f['id']===$scope)$label='Barang pada '.$f['name'];}
        $items=$this->store->listInventory($filter);$stats=$this->statsFromItems($items,$global);$rows=$global['facility_breakdown']??[];
        $occ=['yard_used'=>0,'yard_capacity'=>0,'shed_used'=>0,'shed_capacity'=>0];foreach($facilities as $f)if($facilityId===''||$f['id']===$facilityId){foreach($occ as $key=>$v)$occ[$key]+=(float)($f[$key]??0);}
        $data=$this->baseData($r)+['Title'=>'Dashboard','Subtitle'=>'Ringkasan operasional inventory dan penyelesaian barang','Active'=>'dashboard','Stats'=>$stats,'Facilities'=>$facilities,'DashboardRows'=>$rows,'DashboardOccupancy'=>$occ,'DashboardScope'=>$facilityId===''?'Gabungan seluruh TPP':$this->facilityName($facilities,$facilityId),'FacilityID'=>$facilityId,'DashboardInventoryScope'=>$scope,'DashboardInventoryLabel'=>$label,'CanEditCapacity'=>Domain::can($session,'dashboard.capacity.manage'),'Performance'=>$this->performance($this->store->listEvents(),(string)$r->query('performance_from'),(string)$r->query('performance_to')),'PerformanceOpen'=>(string)$r->query('performance')==='1'];
        return Response::html($this->view->render('dashboard',$data));
    }
    private function capacity(Request $r):Response{$this->store->updateFacilityCapacity((string)$r->route('id'),$this->number($r->input('yard_capacity')),$this->number($r->input('shed_capacity')));return$this->back($r,'Kapasitas TPP berhasil diperbarui.');}

    private function inventory(Request $r):Response
    {
        $session=$this->session($r);$history=$this->bool($r->query('history'));$page=max(1,(int)$r->query('page',1));$pageSize=$this->pageSize($r->query('page_size',20));$filter=['allowed_types'=>Domain::allowedTypes($session),'type'=>strtoupper((string)$r->query('type')),'facility_id'=>(string)$r->query('tpp'),'status'=>(string)$r->query('status'),'sort'=>(string)$r->query('sort','newest'),'query'=>(string)$r->query('q'),'limit'=>$pageSize,'offset'=>($page-1)*$pageSize];if($history)$filter['only_inactive']=true;
        $total=$this->store->countInventory($filter);$items=$this->store->listInventory($filter);$allActive=$this->store->listInventory(['allowed_types'=>Domain::allowedTypes($session),'limit'=>50000]);$groups=$this->inventoryGroups($allActive);
        $data=$this->baseData($r)+['Title'=>$history?'History Inventory':'Inventory','Subtitle'=>$history?'Riwayat barang yang telah keluar atau selesai':'Kelola BTD, BDN, BMMN, dan barang titipan','Active'=>'inventory','History'=>$history,'Items'=>$items,'EligibleItems'=>$allActive,'Facilities'=>$this->store->facilities(),'Query'=>(string)$r->query('q'),'FacilityID'=>(string)$r->query('tpp'),'InventoryType'=>strtoupper((string)$r->query('type')),'Status'=>(string)$r->query('status'),'Sort'=>(string)$r->query('sort','newest'),'Pagination'=>$this->pagination($r,$page,$pageSize,$total),'InventoryActions'=>$this->allowedInventoryActions($session),'CanCreateBTD'=>Domain::can($session,'inventory.create.btd'),'CanCreateBDN'=>Domain::can($session,'inventory.create.bdn'),'CanCreateTitipan'=>Domain::can($session,'inventory.create.titipan'),'CanCreateInventory'=>count(array_filter(['inventory.create.btd','inventory.create.bdn','inventory.create.titipan'],fn($p)=>Domain::can($session,$p)))>0,'CanRunInventoryActions'=>count($this->allowedInventoryActions($session))>0,'ResearchRequestGroups'=>$groups['research'],'CensusTargetGroups'=>$groups['physical'],'RelocationTargetGroups'=>$groups['physical']];
        return Response::html($this->view->render('inventory',$data));
    }
    private function createInventory(Request $r):Response
    {
        $type=strtoupper((string)$r->input('item_type'));$permission=Domain::createPermission($type);if($permission===''||!Domain::can($this->session($r),$permission))throw new ApiException('Anda tidak memiliki akses input jenis inventory tersebut.',403);
        $common=$this->formMap($r,['bl_no','bl_date','manifest_no','manifest_date','manifest_position','determination_no','determination_date','category','entrusted_category','source_office','owner_name','owner_address','origin_warehouse','facility_id','location','load_type','estimated_volume_m3']);$common['type']=$type;$common['at_tpp']=((string)$r->input('at_tpp'))==='sudah';$common['actor']=$this->actor($r);$common['document_id']=$this->optionalDocument($r);
        $inputs=[];$load=strtoupper((string)$r->input('load_type'));
        if($load==='FCL'){
            $containers=$this->jsonArray($r->input('containers_json'));foreach($containers as $ci=>$container){$goods=(array)($container['goods']??[]);foreach($goods as $gi=>$g)$inputs[]=array_merge($common,(array)$g,['load_type'=>'FCL','container_no'=>$container['number']??'','container_size'=>$container['size']??'','physical_unit_id'=>preg_replace('/\W/','',(string)($container['number']??'')),'occupancy_primary'=>$gi===0,'reference_no'=>(string)$common['determination_no'].'/'.str_pad((string)($ci+1),2,'0',STR_PAD_LEFT).'-'.str_pad((string)($gi+1),2,'0',STR_PAD_LEFT)]);}
        }else{
            $goods=$this->jsonArray($r->input('lcl_goods_json'));foreach($goods as $gi=>$g)$inputs[]=array_merge($common,(array)$g,['load_type'=>'LCL','estimated_volume_m3'=>$this->number($r->input('estimated_volume_m3')),'physical_unit_id'=>(string)$common['determination_no'].'-LCL','occupancy_primary'=>$gi===0,'reference_no'=>(string)$common['determination_no'].'/'.str_pad((string)($gi+1),2,'0',STR_PAD_LEFT)]);
        }
        if(!$inputs){$inputs[]=$common+['description'=>(string)$r->input('description'),'item_kind'=>(string)$r->input('item_kind'),'quantity'=>$this->number($r->input('quantity')),'unit'=>(string)$r->input('unit'),'goods_value'=>$this->money($r->input('goods_value'))];}
        if(count($inputs)>1000)throw new ApiException('Maksimal 1.000 baris per penyimpanan.',422);$this->store->createInventories($inputs);return$this->back($r,count($inputs).' barang berhasil dicatat.','/inventory');
    }
    private function importInventory(Request $r):Response
    {
        $file=$r->files['excel_file']??null;if(!is_array($file)||($file['error']??UPLOAD_ERR_NO_FILE)!==UPLOAD_ERR_OK)throw new ApiException('Pilih file XLSX yang valid.',422);if((int)$file['size']>5*1024*1024)throw new ApiException('Ukuran file Excel maksimal 5 MB.',422);
        $rows=Xlsx::read((string)$file['tmp_name'],1002);if(count($rows)<2)throw new ApiException('File Excel tidak memiliki data.',422);$headers=array_map([$this,'normalizeHeader'],$rows[0]);$type=strtoupper((string)$r->input('item_type'));if(!Domain::can($this->session($r),Domain::createPermission($type)))throw new ApiException('Tidak memiliki akses import jenis ini.',403);$inputs=[];
        foreach(array_slice($rows,1) as $line=>$values){if(count(array_filter($values,fn($v)=>trim((string)$v)!==''))===0)continue;$row=[];foreach($headers as $i=>$header)if($header!=='')$row[$header]=$values[$i]??'';$mapped=$this->mapImportRow($row,$type,$line+2);$mapped['actor']=$this->actor($r);$inputs[]=$mapped;}
        if(!$inputs)throw new ApiException('Tidak ada baris data yang dapat diimpor.',422);if(count($inputs)>1000)throw new ApiException('Maksimal 1.000 baris data.',422);$this->store->createInventories($inputs);return$this->back($r,count($inputs).' baris Excel berhasil diimpor.','/inventory');
    }
    private function inventoryEvent(Request $r,bool $bulk):Response
    {
        $code=(string)$r->input('event_code');$permission=Domain::actionPermission($code);if($permission===''||!Domain::can($this->session($r),$permission))throw new ApiException('Anda tidak memiliki hak akses action ini.',403);$ids=$bulk?$this->values($r->input('inventory_ids')):[(string)$r->route('id')];if(!$ids&&$code!=='penelitian_pfpd')throw new ApiException('Pilih minimal satu barang.',422);
        $base=$this->formMap($r,['document_no','document_date','notes','target_facility_id','allocation_type','exit_type']);$base['actor']=$this->actor($r);$base['document_id']=$this->optionalDocument($r);$base['code']=$code;
        if($code==='pencacahan'){
            foreach($this->jsonArray($r->input('census_results_json')) as $target){$this->store->applyCensus((string)($target['target_id']??''),(array)($target['lines']??[]),$base);}return$this->back($r,'Hasil pencacahan berhasil disimpan.','/inventory');
        }
        if($code==='pindah_bongkar_kontainer'){
            $payload=$this->jsonArray($r->input('container_relocation_json'));foreach($payload as $target)$this->store->relocateLoad((string)($target['target_id']??$target['inventory_id']??($ids[0]??'')),(array)($target['allocations']??[$target]),$base);return$this->back($r,'Bongkar/muat kontainer berhasil disimpan.','/inventory');
        }
        if($code==='penelitian_pfpd'){
            foreach($this->jsonArray($r->input('pfpd_results_json')) as $result){$this->store->addInventoryEvent((string)$result['inventory_id'],array_merge($base,$result,['goods_value'=>$this->money($result['goods_value']??0),'restriction_status'=>$result['is_restricted']??'tidak']));}return$this->back($r,'Hasil penelitian PFPD berhasil disimpan.','/inventory');
        }
        foreach($ids as $id)$this->store->addInventoryEvent($id,$base);return$this->back($r,count($ids).' barang berhasil diperbarui.','/inventory');
    }
    private function deleteInventory(Request $r):Response{$this->store->deleteInventory((string)$r->route('id'),$this->actor($r));return$this->back($r,'Barang dan jejak terkait berhasil dihapus.','/inventory');}

    private function processPage(Request $r):Response
    {
        [$type,$title,$singular,$viewPermission,$managePermission]=$this->processMeta((string)$r->route('type'));$session=$this->session($r);if(!Domain::can($session,$viewPermission))throw new ApiException('Akses proses tidak diberikan.',403);$history=$this->bool($r->query('history'));$page=max(1,(int)$r->query('page',1));$size=$this->pageSize($r->query('page_size',20));$filter=['type'=>$type,'facility_id'=>(string)$r->query('tpp'),'status'=>(string)$r->query('status'),'query'=>(string)$r->query('q'),'sort'=>(string)$r->query('sort','newest'),'allowed_types'=>Domain::allowedTypes($session),'limit'=>$size,'offset'=>($page-1)*$size,'include_inactive_inventory'=>$history];if($history)$filter['only_inactive_inventory']=true;
        $total=$this->store->countDispositions($filter);$processes=$this->store->listDispositions($filter);$eligible=$this->store->listInventory(['allowed_types'=>Domain::allowedTypes($session),'limit'=>50000]);$dashboard=$this->store->processDashboard($type,(int)date('Y'),Domain::allowedTypes($session));$groups=$this->auctionScheduleGroups($this->store->listDispositions(['type'=>'lelang','include_inactive_inventory'=>true,'limit'=>50000]));
        $data=$this->baseData($r)+['Title'=>$title,'Subtitle'=>'Kelola tahapan dan dokumen '.$title,'Active'=>$type,'ProcessType'=>$type,'ProcessTitle'=>$title,'ProcessSingular'=>$singular,'History'=>$history,'Processes'=>$processes,'CandidateProcesses'=>$processes,'EligibleItems'=>$eligible,'ProcessActions'=>Domain::actionsFor($type),'CanManage'=>Domain::can($session,$managePermission),'Facilities'=>$this->store->facilities(),'Query'=>(string)$r->query('q'),'FacilityID'=>(string)$r->query('tpp'),'Status'=>(string)$r->query('status'),'Sort'=>(string)$r->query('sort','newest'),'Pagination'=>$this->pagination($r,$page,$size,$total),'ProcessDashboard'=>$dashboard,'AuctionDashboard'=>$type==='lelang'?$dashboard:[],'DestructionDashboard'=>$type==='musnah'?$dashboard:[],'GrantDashboard'=>$type==='hibah'?$dashboard:[],'AuctionScheduleGroups'=>$groups];return Response::html($this->view->render('process',$data));
    }
    private function processAction(Request $r):Response
    {
        [$type,,,$view,$manage]=$this->processMeta((string)$r->route('type'));if(!Domain::can($this->session($r),$manage))throw new ApiException('Anda tidak memiliki akses kelola proses.',403);$code=(string)$r->input('event_code');$base=$this->formMap($r,['document_no','document_date','notes','execution_start_date','execution_end_date','transfer_type','allocation_target']);$base['actor']=$this->actor($r);$base['document_id']=$this->optionalDocument($r);$base['code']=$code;
        $ids=$this->values($r->input('process_ids'));$inventoryIds=$this->values($r->input('inventory_ids'));
        if($code==='kep_lelang'||$code==='kep_musnah'||$code==='ba_serah_terima')foreach($inventoryIds as $inventoryId){$p=$this->store->createDisposition($inventoryId,$type,$this->actor($r),(string)$r->input('notes'));if($code!==($type==='lelang'?'kep_lelang':($type==='musnah'?'kep_musnah':'ba_serah_terima')))$this->store->addDispositionEvent((string)$p['id'],$base);}
        elseif($code==='kep_htl'&&$r->input('htl_results_json'))foreach($this->jsonArray($r->input('htl_results_json')) as $row)$this->store->addDispositionEvent((string)$row['process_id'],array_merge($base,['htl_value'=>$this->money($row['htl_value']??0)]));
        elseif($code==='selesai_lelang'&&$r->input('auction_results_json'))foreach($this->jsonArray($r->input('auction_results_json')) as $row)$this->store->addDispositionEvent((string)$row['process_id'],array_merge($base,['auction_outcome'=>$row['outcome']??'','sale_value'=>$this->money($row['sale_value']??0)]));
        else foreach($ids as $id)$this->store->addDispositionEvent($id,array_merge($base,['htl_value'=>$this->money($r->input('htl_value')),'sale_value'=>$this->money($r->input('sale_value')),'destruction_cost'=>$this->money($r->input('destruction_cost')),'auction_outcome'=>(string)$r->input('auction_outcome')]));
        return$this->back($r,'Tahapan proses berhasil disimpan.','/proses/'.$type);
    }

    private function reconciliation(Request $r): Response
    {
        $session = $this->session($r);
        $tab = (string) $r->query('tab', 'rekonsiliasi');
        if ($tab !== 'perubahan-data') {
            $tab = 'rekonsiliasi';
        }
        $items = $this->store->listInventory([
            'allowed_types' => Domain::allowedTypes($session),
            'include_inactive' => true,
            'limit' => 50000,
        ]);
        $records = $this->filterReconciliationsForSession($this->store->listReconciliations(), $session);
        [$reconciliations, $corrections] = $this->splitReconciliations($records);
        $data = array_merge($this->baseData($r), [
            'Title' => 'Rekonsiliasi Barang',
            'Subtitle' => 'Rekonsiliasi kondisi fisik dan audit perubahan data',
            'Active' => 'reconciliation',
            'ReconciliationTab' => $tab,
            'Items' => $items,
            'EligibleItems' => $items,
            'Reconciliations' => $reconciliations,
            'DataCorrections' => $corrections,
            'DataCorrectionRows' => $this->correctionRows($corrections),
            'Facilities' => $this->store->facilities(),
            'CanManage' => Domain::can($session, 'reconciliation.manage'),
        ]);
        return Response::html($this->view->render('reconciliation', $data));
    }
    private function reconcile(Request $r):Response
    {
        $type=(string)$r->input('reconciliation_type');$doc=$this->optionalDocument($r);if($type==='data_correction'){
            $item=$this->jsonObject($r->input('correction_item_json'));$original=$this->store->getInventory((string)($item['id']??$r->input('inventory_id')));$changes=[];foreach($item as $key=>$value)if(array_key_exists($key,$original)&&$original[$key]!=$value&&!in_array($key,['id','created_at','updated_at','created_by'],true))$changes[$key]=$value;
            if(!$changes)throw new ApiException('Tidak ada perubahan data yang terdeteksi.',422);$this->store->correctInventory(['p_inventory_id'=>$original['id'],'p_changes'=>$changes,'p_events'=>$this->jsonArray($r->input('correction_events_json')),'p_processes'=>$this->jsonArray($r->input('correction_processes_json')),'p_reason'=>(string)$r->input('correction_reason'),'p_actor'=>$this->actor($r),'p_document_id'=>$doc?:null]);return$this->back($r,'Perubahan data barang berhasil disimpan dengan jejak audit.','/rekonsiliasi?tab=perubahan-data');
        }
        $input=['type'=>$type,'inventory_id'=>(string)$r->input('inventory_id'),'notes'=>(string)$r->input('notes'),'actor'=>$this->actor($r),'document_id'=>$doc,'new_item'=>$this->formMap($r,['item_type','determination_no','determination_date','manifest_no','manifest_date','manifest_position','description','item_kind','quantity','unit','goods_value','category','entrusted_category','source_office','origin_warehouse','facility_id','location','load_type','container_no','container_size','estimated_volume_m3','initial_status_code'])];$input['new_item']['type']=strtoupper((string)$input['new_item']['item_type']);$input['new_item']['at_tpp']=(string)$input['new_item']['facility_id']!=='';$this->store->reconcile($input);return$this->back($r,'Rekonsiliasi berhasil disimpan.','/rekonsiliasi');
    }

    private function reports(Request $r):Response
    {
        $preset=trim((string)$r->query('preset'));
        $page=max(1,(int)$r->query('page',1));$size=$this->pageSize($r->query('page_size',20));
        $common=$this->baseData($r)+['Title'=>'Pelaporan','Subtitle'=>'Filter, tinjau, dan ekspor data operasional LIVIRA','Active'=>'reports','Facilities'=>$this->store->facilities(),'ReportPerformance'=>false,'ReportReconciliation'=>false,'ReportDataCorrection'=>false,'ReportBTD'=>false,'Items'=>[],'Reconciliations'=>[],'DataCorrections'=>[],'DataCorrectionRows'=>[],'BTDReportRows'=>[]];
        if($preset==='performance'){
            $performance=$this->performanceReport($r,(string)$r->query('date_from'),(string)$r->query('date_to'));
            $report=['Preset'=>'performance','Title'=>'Performa kinerja','Description'=>'Jumlah penyelesaian dan rata-rata waktu proses berdasarkan rentang tanggal selesai.','ExportURL'=>$performance['ExportURL'],'CSVExportURL'=>'','ExcelExportURL'=>$performance['ExportURL']];
            return Response::html($this->view->render('reports',$common+['Report'=>$report,'ReportPerformance'=>true,'Performance'=>$performance,'ReportTotal'=>$performance['TotalCompleted'],'Pagination'=>$this->pagination($r,1,$size,$performance['TotalCompleted'])]));
        }
        if(in_array($preset,['reconciliation','data_correction'],true)){
            if(!Domain::can($this->session($r),'reconciliation.view'))throw new ApiException('Anda tidak memiliki hak akses laporan rekonsiliasi.',403);
            $records=$this->filterReconciliationsForSession($this->store->listReconciliations(),$this->session($r));[$regular,$corrections]=$this->splitReconciliations($records);
            if($preset==='reconciliation'){
                $total=count($regular);$paged=array_slice($regular,($page-1)*$size,$size);$report=$this->reportOptions($r,$preset,'Rekap rekonsiliasi','Penambahan atau pengeluaran inventory berdasarkan perbandingan catatan aplikasi dan kondisi fisik di lapangan.');
                return Response::html($this->view->render('reports',$common+['Report'=>$report,'ReportReconciliation'=>true,'Reconciliations'=>$paged,'ReportTotal'=>$total,'Pagination'=>$this->pagination($r,$page,$size,$total)]));
            }
            $flat=$this->correctionRows($corrections);$total=count($flat);$paged=array_slice($flat,($page-1)*$size,$size);$report=$this->reportOptions($r,$preset,'Rekap perubahan data barang','Audit rinci data yang diubah beserta nilai sebelum, nilai sesudah, alasan, waktu, dan petugas.');
            return Response::html($this->view->render('reports',$common+['Report'=>$report,'ReportDataCorrection'=>true,'DataCorrections'=>$corrections,'DataCorrectionRows'=>$paged,'ReportTotal'=>$total,'ReportTransactionTotal'=>count($corrections),'Pagination'=>$this->pagination($r,$page,$size,$total)]));
        }
        [$filter,$items,$report]=$this->reportData($r);
        if($preset==='btd'){
            $rows=$this->btdRows($items);$total=count($rows);$paged=array_slice($rows,($page-1)*$size,$size);
            return Response::html($this->view->render('reports',$common+['Report'=>$report,'ReportBTD'=>true,'BTDReportRows'=>$paged,'ReportTotal'=>$total,'ReportTotalValue'=>array_sum(array_map(fn($i)=>(float)($i['goods_value']??0),$items)),'Pagination'=>$this->pagination($r,$page,$size,$total),'FacilityID'=>$filter['facility_id']??'','Status'=>$filter['status']??'']));
        }
        $total=count($items);$paged=array_slice($items,($page-1)*$size,$size);
        return Response::html($this->view->render('reports',$common+['Report'=>$report,'Items'=>$paged,'ReportTotal'=>$total,'ReportActive'=>count(array_filter($items,fn($i)=>!empty($i['is_active']))),'ReportClosed'=>count(array_filter($items,fn($i)=>empty($i['is_active']))),'ReportTotalValue'=>array_sum(array_map(fn($i)=>(float)($i['goods_value']??0),$items)),'ReportAtTPP'=>count(array_filter($items,fn($i)=>!empty($i['at_tpp']))),'ReportTransactionTotal'=>$total,'Pagination'=>$this->pagination($r,$page,$size,$total),'FacilityID'=>$filter['facility_id']??'','InventoryType'=>$filter['type']??'','Status'=>$filter['status']??'']));
    }
    private function exportReport(Request $r,string $format):Response
    {
        $preset=trim((string)$r->query('preset'));
        if($preset==='performance')return $this->exportPerformance($r);
        if(in_array($preset,['reconciliation','data_correction'],true)&&!Domain::can($this->session($r),'reconciliation.view'))throw new ApiException('Anda tidak memiliki hak akses laporan rekonsiliasi.',403);
        if($preset==='reconciliation'){
            $records=$this->filterReconciliationsForSession($this->store->listReconciliations(),$this->session($r));[$records]=$this->splitReconciliations($records);$headers=['Tanggal','Jenis Rekonsiliasi','Tindakan','Referensi Inventory','Jenis Barang','Status Sebelumnya','Status Hasil','Catatan','Petugas'];$rows=[];
            foreach($records as $x)$rows[]=[(string)($x['created_at']??''),(string)($x['reconciliation_type']??''),($x['reconciliation_type']??'')==='recorded_not_found'?'Dikeluarkan dari inventory aktif':'Ditambahkan ke inventory',(string)($x['inventory_reference']??''),(string)($x['inventory_type']??''),(string)($x['previous_status_label']??''),(string)($x['result_status_label']??''),(string)($x['notes']??''),(string)($x['actor']??'')];
            return $this->tableExport($format,$headers,$rows,'livira-rekonsiliasi','Rekonsiliasi');
        }
        if($preset==='data_correction'){
            $records=$this->filterReconciliationsForSession($this->store->listCorrections(),$this->session($r));$flat=$this->correctionRows($records);$headers=['Tanggal','Referensi Inventory','Jenis Barang','Bagian','Konteks','Kolom','Nilai Sebelum','Nilai Sesudah','Alasan','Petugas'];$rows=[];
            foreach($flat as $x){$rec=(array)($x['Record']??[]);$ch=(array)($x['Change']??[]);$rows[]=[(string)($rec['created_at']??''),(string)($rec['inventory_reference']??''),(string)($rec['inventory_type']??''),(string)($ch['section']??''),(string)($ch['context']??''),(string)($ch['field']??''),(string)($ch['before']??''),(string)($ch['after']??''),(string)($rec['correction_reason']??$rec['notes']??''),(string)($rec['actor']??'')];}
            return $this->tableExport($format,$headers,$rows,'livira-perubahan-data','Perubahan Data');
        }
        [$filter,$items]=$this->reportData($r);
        if($preset==='btd'){
            $headers=['Nomor BTD','Tanggal BTD','Nomor BL','Tanggal BL','Nomor Manifest','Tanggal Manifest','Pos Manifest','Jenis Muatan','TPS Asal','TPP','Status Lokasi','Kontainer / LCL','Jumlah Kontainer','Uraian, Jenis, Kondisi, dan Jumlah Barang','Jumlah Rincian Barang','Total Nilai Barang','Pemilik / Shipper / Consignee','Status Barang','Status Inventory'];$rows=[];
            foreach($this->btdRows($items) as $x)$rows[]=array_values($x);
            return $this->tableExport($format,$headers,$rows,'livira-btd','Laporan BTD');
        }
        $headers=['Jenis','Nomor Referensi','Nomor Penetapan/Dokumen','Tanggal Penetapan','Nomor BL','Tanggal BL','Nomor Manifest','Tanggal Manifest','Pos Manifest','Nomor Kontainer','Ukuran','Muatan','Uraian Barang','Jenis Barang','Kondisi','Jumlah','Satuan','Nilai Barang','TPS Asal','TPP','Status Lokasi','Status Barang','Status Inventory'];$rows=[];
        foreach($items as $i)$rows[]=[(string)($i['item_type']??''),(string)($i['reference_no']??''),(string)($i['determination_no']??''),(string)($i['determination_date']??''),(string)($i['bl_no']??''),(string)($i['bl_date']??''),(string)($i['manifest_no']??''),(string)($i['manifest_date']??''),(string)($i['manifest_position']??''),(string)($i['container_no']??''),(string)($i['container_size']??''),(string)($i['load_type']??''),(string)($i['description']??''),(string)($i['item_kind']??''),(string)($i['goods_condition']??''),(float)($i['quantity']??0),(string)($i['unit']??''),(int)($i['goods_value']??0),(string)($i['origin_warehouse']??''),(string)($i['facility_name']??''),(string)($i['location_status']??''),(string)($i['status_label']??''),!empty($i['is_active'])?'Aktif':'Selesai'];
        return $this->tableExport($format,$headers,$rows,'livira-laporan','Laporan LIVIRA');
    }
    private function exportPerformance(Request $r):Response
    {
        $performance=$this->performanceReport($r,(string)$r->query('date_from'),(string)$r->query('date_to'));
        $summaryHeaders=['Indikator','Jumlah Dokumen','Rata-rata Durasi (Jam)','Sampel Durasi Valid','Keterangan'];
        $summaryRows=[];
        foreach($performance['Metrics'] as $metric){
            $summaryRows[]=[
                (string)($metric['Label']??''),
                (int)($metric['Count']??0),
                round((float)($metric['AverageHours']??0),2),
                (int)($metric['DurationSamples']??0),
                (string)($metric['Description']??''),
            ];
        }
        $summaryRows[]=['Total dokumen penyelesaian',(int)($performance['TotalCompleted']??0),'','',(string)($performance['PeriodLabel']??'')];

        $detailHeaders=['Kategori','Dokumen Selesai','Tanggal Selesai','Dokumen Awal/Request','Tanggal Awal/Request','Durasi Jam','Durasi Valid','Jumlah Barang'];
        $detailRows=[];
        foreach($performance['Details'] as $detail){
            $detailRows[]=[
                (string)($detail['MetricLabel']??''),
                (string)($detail['CompletionDocument']??''),
                (string)($detail['CompletionDate']??''),
                (string)($detail['StartDocument']??''),
                (string)($detail['StartDate']??''),
                round((float)($detail['DurationHours']??0),2),
                !empty($detail['DurationValid'])?'Ya':'Tidak',
                (int)($detail['InventoryCount']??0),
            ];
        }
        $content=Xlsx::writeSheets([
            ['name'=>'Ringkasan','headers'=>$summaryHeaders,'rows'=>$summaryRows],
            ['name'=>'Rincian','headers'=>$detailHeaders,'rows'=>$detailRows],
        ]);
        return Response::file($content,'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet','livira-performa-'.date('Ymd-His').'.xlsx');
    }
    private function tableExport(string $format,array $headers,array $rows,string $base,string $sheet):Response{$name=$base.'-'.date('Ymd-His');if($format==='csv')return Response::file(Xlsx::csv($headers,$rows),'text/csv; charset=utf-8',$name.'.csv');if($format==='xls')return Response::file($this->htmlTable($headers,$rows),'application/vnd.ms-excel; charset=utf-8',$name.'.xls');return Response::file(Xlsx::write($headers,$rows,$sheet),'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',$name.'.xlsx');}

    private function searchPage(Request $r):Response{$q=trim((string)$r->query('q'));$items=$q===''?[]:$this->store->listInventory(['query'=>$q,'include_inactive'=>true,'allowed_types'=>Domain::allowedTypes($this->session($r)),'limit'=>200]);return Response::html($this->view->render('search',$this->baseData($r)+['Title'=>'Pencarian Detail Barang','Subtitle'=>'Cari seluruh data barang dan buka timeline aktivitas','Active'=>'search','Query'=>$q,'SearchPerformed'=>$q!=='','Items'=>$items,'Search'=>$items]));}
    private function inventoryDetail(Request $r):Response{$item=$this->accessibleInventory($r,(string)$r->route('id'));return Response::json(['item'=>$item]);}
    private function inventoryTimeline(Request $r):Response{$id=(string)$r->route('id');$item=$this->accessibleInventory($r,$id);return Response::json(['item'=>$item,'events'=>$this->store->timeline($id),'processes'=>$this->store->listDispositions(['inventory_id'=>$id,'include_inactive_inventory'=>true,'limit'=>100])]);}
    private function processTimeline(Request $r):Response{$id=(string)$r->route('id');$process=$this->store->getDisposition($id);$type=(string)($process['inventory_type']??($process['inventory']['item_type']??''));if($type!==''&&!in_array(strtoupper($type),Domain::allowedTypes($this->session($r)),true)&&($this->session($r)['Role']??'')!=='admin')throw new ApiException('Data proses tidak dapat diakses.',403);$permission=match((string)($process['disposition_type']??'')){'lelang'=>'auction.view','musnah'=>'destruction.view','hibah'=>'grant.view',default=>''};if($permission!==''&&!Domain::can($this->session($r),$permission))throw new ApiException('Data proses tidak dapat diakses.',403);return Response::json(['process'=>$process,'events'=>$this->store->processTimeline($id)]);}
    private function downloadDocument(Request $r):Response{$id=(string)$r->route('id');if(($this->session($r)['Role']??'')!=='admin'){$links=$this->store->documentAccess($id);$allowed=false;foreach($links as $link){$item=(array)($link['inventory']??[]);$type=strtoupper((string)($item['item_type']??''));if($type===''||!in_array($type,Domain::allowedTypes($this->session($r)),true))continue;$processPermission=match((string)($link['disposition_type']??'')){'lelang'=>'auction.view','musnah'=>'destruction.view','hibah'=>'grant.view',default=>''};if($processPermission===''||Domain::can($this->session($r),$processPermission)){$allowed=true;break;}}if(!$allowed)throw new ApiException('Dokumen tidak dapat diakses.',403);}[$meta,$content]=$this->store->getDocument($id);return Response::file($content,(string)($meta['mime_type']??'application/octet-stream'),(string)($meta['file_name']??'dokumen'));}
    private function accessibleInventory(Request $r,string $id):array{$item=$this->store->getInventory($id);if(($this->session($r)['Role']??'')!=='admin'&&!in_array(strtoupper((string)($item['item_type']??'')),Domain::allowedTypes($this->session($r)),true))throw new ApiException('Data inventory tidak dapat diakses.',403);return$item;}

    private function adminUsers(Request $r):Response{$users=$this->store->listUsers();return Response::html($this->view->render('admin',$this->baseData($r)+['Title'=>'Persetujuan Pendaftaran','Subtitle'=>'Verifikasi akun dan tetapkan role pengguna','Active'=>'admin-users','AdminSection'=>'users','Users'=>$users,'Roles'=>$this->store->listRoles(false),'PendingUsers'=>count(array_filter($users,fn($u)=>($u['approval_status']??'')==='pending')),'VerifiedPendingUsers'=>count(array_filter($users,fn($u)=>($u['approval_status']??'')==='pending'&&!empty($u['email_verified'])))]));}
    private function approveUser(Request $r):Response{$this->store->approveUser((string)$r->route('id'),(string)$r->input('role_id'),$this->actor($r));return$this->back($r,'Pendaftaran berhasil disetujui.','/admin/pendaftaran');}
    private function rejectUser(Request $r):Response{$this->store->rejectUser((string)$r->route('id'),(string)$r->input('reason'),$this->actor($r));return$this->back($r,'Pendaftaran ditolak.','/admin/pendaftaran');}
    private function updateUserRole(Request $r):Response{$this->store->updateUserRole((string)$r->route('id'),(string)$r->input('role_id'),$this->actor($r));return$this->back($r,'Role pengguna berhasil diperbarui.','/admin/pendaftaran');}
    private function deleteUser(Request $r):Response{$this->store->deleteUser((string)$r->route('id'));return$this->back($r,'Pengguna berhasil dihapus dari aplikasi dan autentikasi.','/admin/pendaftaran');}
    private function adminRoles(Request $r):Response{return Response::html($this->view->render('admin',$this->baseData($r)+['Title'=>'Role & Hak Akses','Subtitle'=>'Atur hak akses granular setiap role','Active'=>'admin-roles','AdminSection'=>'roles','Roles'=>$this->store->listRoles(),'PermissionDefinitions'=>Domain::PERMISSIONS]));}
    private function createRole(Request $r):Response{$this->store->createRole(['name'=>(string)$r->input('name'),'description'=>(string)$r->input('description'),'permissions'=>$this->values($r->input('permissions')),'actor'=>$this->actor($r)]);return$this->back($r,'Role berhasil dibuat.','/admin/roles');}
    private function updateRole(Request $r):Response{$this->store->updateRole((string)$r->route('id'),['name'=>(string)$r->input('name'),'description'=>(string)$r->input('description'),'permissions'=>$this->values($r->input('permissions'))]);return$this->back($r,'Role berhasil diperbarui.','/admin/roles');}
    private function roleStatus(Request $r):Response{$this->store->setRoleActive((string)$r->route('id'),$this->bool($r->input('active')));return$this->back($r,'Status role berhasil diperbarui.','/admin/roles');}
    private function deleteRole(Request $r):Response{$roles=$this->store->listRoles();foreach($roles as $role)if(($role['id']??'')===$r->route('id')&&(int)($role['assigned_users']??0)>0)throw new ApiException('Role tidak dapat dihapus karena masih digunakan oleh pengguna.',409);$this->store->deleteRole((string)$r->route('id'));return$this->back($r,'Role kosong berhasil dihapus.','/admin/roles');}
    private function adminParameters(Request $r):Response{return Response::html($this->view->render('admin',$this->baseData($r)+['Title'=>'Parameter Sistem','Subtitle'=>'Kelola pilihan dropdown tanpa menampilkan kode internal','Active'=>'admin-parameters','AdminSection'=>'parameters','Parameters'=>$this->store->parameterOptions()]));}
    private function createParameter(Request $r):Response{$this->store->createParameter(['group_code'=>(string)$r->input('group_code'),'code'=>(string)$r->input('code'),'label'=>(string)$r->input('label'),'sort_order'=>(int)$r->input('sort_order',999),'applies_to'=>$this->values($r->input('applies_to')),'actor'=>$this->actor($r)]);return$this->back($r,'Parameter berhasil dibuat.','/admin/parameters');}
    private function updateParameter(Request $r):Response{$this->store->updateParameter((string)$r->route('id'),['label'=>(string)$r->input('label'),'sort_order'=>(int)$r->input('sort_order',999),'applies_to'=>$this->values($r->input('applies_to'))]);return$this->back($r,'Parameter berhasil diperbarui.','/admin/parameters');}
    private function parameterStatus(Request $r):Response{$this->store->setParameterActive((string)$r->route('id'),$this->bool($r->input('active')));return$this->back($r,'Status parameter berhasil diperbarui.','/admin/parameters');}

    private function auditRequest(Request $r,Response $response):void
    {
        $isExport=$r->method==='GET'&&in_array($r->path,['/pelaporan.csv','/pelaporan.xlsx','/pelaporan.xls','/pelaporan/performa.xlsx'],true);
        if($r->method!=='POST'&&!$isExport)return;
        if(in_array($r->path,['/session/activity','/captcha/new'],true))return;
        $session=(array)($r->attributes['session']??[]);
        try{$this->store->audit(['actor_subject'=>(string)($session['Subject']??''),'actor_name'=>(string)($session['DisplayName']??'Pengguna anonim'),'action'=>$isExport?'export':'request_mutation','entity_type'=>'http_request','entity_id'=>(string)($r->route('id')??''),'outcome'=>$response->status>=400?'failed':'success','ip_address'=>$r->ip(),'user_agent'=>$r->userAgent(),'request_id'=>bin2hex(random_bytes(8)),'metadata'=>['method'=>$r->method,'path'=>$r->path,'status'=>$response->status]]);}catch(\Throwable){}
    }

    private function protected(Request $r,callable $next):Response
    {
        $session=$this->sessions->read();if(!$session)return$r->acceptsJson()?Response::json(['error'=>'Sesi tidak valid atau telah berakhir.'],401):Response::redirect('/login?error='.rawurlencode('Sesi berakhir. Silakan login kembali.'));
        if(($session['Role']??'')!=='admin'&&str_starts_with((string)($session['Subject']??''),'user:')){try{$user=$this->store->userByAuthId(substr((string)$session['Subject'],5));if(($user['approval_status']??'')!=='approved')return Response::redirect('/login?error='.rawurlencode('Akses akun tidak lagi aktif.'))->withCookie($this->sessions->clearCookie());$session['DisplayName']=$user['name']??$session['DisplayName'];$session['RoleID']=$user['role_id']??'';$session['RoleName']=$user['role_name']??'';$session['Permissions']=Domain::normalizePermissions((array)($user['permissions']??[]));}catch(\Throwable){return Response::redirect('/login?error='.rawurlencode('Akun aplikasi tidak ditemukan.'))->withCookie($this->sessions->clearCookie());}}
        $session=$this->sessions->touch($session);$r->attributes['session']=$session;$response=$next($r);return$response->withCookie($this->sessions->cookie($session));
    }
    private function csrf(Request $r,callable $next):Response{if(!$this->sessions->csrfValid($this->session($r),(string)$r->input('_csrf')))return$this->errorResponse($r,'Token keamanan form tidak valid. Muat ulang halaman lalu coba kembali.',419);return$next($r);}
    private function permission(Request $r,callable $next,string $permission):Response{if(!Domain::can($this->session($r),$permission))return$this->errorResponse($r,'Anda tidak memiliki hak akses untuk fitur ini.',403);return$next($r);}
    private function anyPermission(Request $r,callable $next,array $permissions):Response{foreach($permissions as $p)if(Domain::can($this->session($r),$p))return$next($r);return$this->errorResponse($r,'Anda tidak memiliki hak akses untuk fitur ini.',403);}
    private function adminOnly(Request $r,callable $next):Response{if(($this->session($r)['Role']??'')!=='admin')return$this->errorResponse($r,'Fitur ini khusus administrator.',403);return$next($r);}
    private function session(Request $r):array{return (array)($r->attributes['session']??[]);}
    private function actor(Request $r):string{return (string)($this->session($r)['DisplayName']??'Pengguna LIVIRA');}

    private function baseData(Request $r):array
    {
        $session=$this->session($r);$parameters=$this->store->parameterOptions('',false);$labels=function(string $group,?string $type=null)use($parameters){$out=[];foreach($parameters as $p){if(($p['group_code']??'')!==$group||empty($p['active']))continue;$applies=trim((string)($p['applies_to']??''));if($type&&$applies!==''&&!in_array($type,array_map('trim',explode(',',$applies)),true))continue;$out[]=(string)($p['label']??'');}return array_values(array_unique($out));};$options=function(string $group)use($parameters){$out=[];foreach($parameters as $p)if(($p['group_code']??'')===$group&&!empty($p['active']))$out[]=['Code'=>$p['code'],'Label'=>$p['label'],'Types'=>$p['applies_to']??''];return$out;};$notifications=['count'=>0,'items'=>[]];try{$notifications=$this->store->notifications(Domain::allowedTypes($session));}catch(\Throwable){}
        return['User'=>$session,'CSRF'=>$session['CSRF']??'','Now'=>gmdate('c'),'IdleTimeoutSeconds'=>$this->config->idleTimeoutSeconds,'Success'=>(string)$r->query('success'),'Error'=>(string)$r->query('error'),'DemoMode'=>$this->config->demoMode,'Notifications'=>$notifications['items']??[],'NotificationCount'=>(int)($notifications['count']??0),'TPSNames'=>$labels('origin_tps'),'BDNCategoryNames'=>$labels('bdn_category'),'EntrustedCategoryNames'=>$labels('entrusted_category'),'ItemKindNames'=>$labels('item_kind'),'GoodsConditionNames'=>$labels('goods_condition'),'AllocationPurposeNames'=>$labels('allocation_purpose'),'UnitNames'=>$labels('unit'),'LoadTypeOptions'=>$options('load_type'),'ExitOptions'=>$options('exit_type'),'TransferTypeOptions'=>$options('transfer_type'),'ContainerSizeOptions'=>[['Code'=>'20','Label'=>"20'"],['Code'=>'40','Label'=>"40'"],['Code'=>'40HC','Label'=>"40' HC"],['Code'=>'45HC','Label'=>"45' HC"]]];
    }
    private function statsFromItems(array $items, array $global): array
    {
        $types = ['BTD' => [], 'BDN' => [], 'BMMN' => [], 'TITIPAN' => []];
        foreach ($items as $item) {
            $type = (string) ($item['item_type'] ?? '');
            if (isset($types[$type])) {
                $types[$type][] = $item;
            }
        }

        $summary = static function (array $rows): array {
            $documents = array_unique(array_filter(array_map(
                static fn(array $row): string => trim((string) ($row['determination_no'] ?? '')),
                $rows
            )));
            $fclUnits = array_unique(array_filter(array_map(
                static fn(array $row): ?string => strtoupper((string) ($row['load_type'] ?? '')) === 'FCL'
                    ? (string) ($row['physical_unit_id'] ?? $row['container_no'] ?? '')
                    : null,
                $rows
            )));
            $lcl = count(array_filter(
                $rows,
                static fn(array $row): bool => strtoupper((string) ($row['load_type'] ?? '')) === 'LCL'
            ));

            return [
                'documents' => count($documents),
                'fcl' => count($fclUnits),
                'lcl' => $lcl,
            ];
        };

        return [
            'active_total' => count($items),
            'btd_total' => count($types['BTD']),
            'bdn_total' => count($types['BDN']),
            'bmmn_total' => count($types['BMMN']),
            'titipan_total' => count($types['TITIPAN']),
            'active_summary' => $summary($items),
            'btd_summary' => $summary($types['BTD']),
            'bdn_summary' => $summary($types['BDN']),
            'bmmn_summary' => $summary($types['BMMN']),
            'titipan_summary' => $summary($types['TITIPAN']),
            'auction_active' => $global['auction_active'] ?? 0,
            'destruction_active' => $global['destruction_active'] ?? 0,
            'grant_active' => $global['grant_active'] ?? 0,
            'recent_events' => $global['recent_events'] ?? [],
            'attention_items' => $global['attention_items'] ?? [],
        ];
    }
    private function performanceReport(Request $r,string $from='',string $to=''):array
    {
        $start=$from!==''?strtotime($from.' 00:00:00'):strtotime(date('Y').'-01-01 00:00:00');
        $end=$to!==''?strtotime($to.' 23:59:59'):strtotime(date('Y').'-12-31 23:59:59');
        if($start>$end){[$start,$end]=[$end,$start];}
        $items=$this->store->listInventory(['include_inactive'=>true,'allowed_types'=>Domain::allowedTypes($this->session($r)),'limit'=>50000]);
        $byId=[];foreach($items as $item)if(($item['item_type']??'')!=='TITIPAN')$byId[(string)$item['id']]=$item;
        $definitions=[
            'auction'=>['Label'=>'Performa lelang','Description'=>'Selesai lelang, dihitung sejak penetapan awal BTD/BDN.'],
            'destruction'=>['Label'=>'Performa musnah','Description'=>'BA Musnah, dihitung sejak penetapan awal BTD/BDN.'],
            'grant'=>['Label'=>'Performa hibah/PSP','Description'=>'BA Serah Terima Hibah/PSP, dihitung sejak penetapan awal BTD/BDN.'],
            'census'=>['Label'=>'Performa cacah','Description'=>'Pencacahan selesai, dihitung sejak penetapan sampai BA Cacah.'],
            'pfpd'=>['Label'=>'Performa penilaian PFPD','Description'=>'Penilaian selesai, dihitung sejak request penelitian PFPD.'],
            'bmmn'=>['Label'=>'Konversi BMMN','Description'=>'Penetapan BMMN dari BTD/BDN, dihitung sejak penetapan awal.'],
        ];
        $groups=[];
        foreach($this->store->listEvents() as $event){$id=(string)($event['inventory_id']??'');if(!isset($byId[$id]))continue;$code=(string)($event['code']??'');$metric=match($code){'selesai_lelang','laku','alokasi_hasil_lelang'=>'auction','ba_musnah'=>'destruction','ba_serah_terima'=>'grant','pencacahan'=>'census','penelitian_pfpd'=>'pfpd','penetapan_bmmn'=>'bmmn',default=>''};if($metric==='')continue;$completionRaw=(string)($event['document_date']??$event['created_at']??'');$completion=strtotime($completionRaw);if(!$completion||$completion<$start||$completion>$end)continue;$item=$byId[$id];$document=trim((string)($event['document_no']??''));$key=$metric.'|'.mb_strtoupper($document).'|'.date('Y-m-d',$completion);if($document==='')$key.='|'.(string)($event['id']??$id);
            $startRaw=(string)($item['determination_date']??$item['created_at']??'');$startDoc=(string)($item['determination_no']??'');if($metric==='pfpd'){$startRaw=(string)($item['research_request_date']??'');$startDoc=(string)($item['research_request_no']??'');}
            $startAt=$startRaw!==''?strtotime($startRaw):false;$g=$groups[$key]??['Metric'=>$metric,'MetricLabel'=>$definitions[$metric]['Label'],'CompletionDocument'=>$document,'CompletionDate'=>date('c',$completion),'StartDocument'=>$startDoc,'StartDate'=>$startAt?date('c',$startAt):null,'DurationHours'=>0.0,'DurationValid'=>false,'InventoryIDs'=>[]];$g['InventoryIDs'][$id]=true;if($startAt&&$startAt<=$completion){$hours=($completion-$startAt)/3600;if(!$g['DurationValid']||$hours<$g['DurationHours']){$g['DurationHours']=$hours;$g['DurationValid']=true;$g['StartDate']=date('c',$startAt);$g['StartDocument']=$startDoc;}}$groups[$key]=$g;
        }
        $details=[];$stats=[];foreach($definitions as $code=>$d)$stats[$code]=['Label'=>$d['Label'],'Count'=>0,'AverageHours'=>0.0,'DurationSamples'=>0,'Description'=>$d['Description'],'_sum'=>0.0];
        foreach($groups as $g){$g['InventoryCount']=count($g['InventoryIDs']);unset($g['InventoryIDs']);$details[]=$g;$stats[$g['Metric']]['Count']++;if($g['DurationValid']){$stats[$g['Metric']]['DurationSamples']++;$stats[$g['Metric']]['_sum']+=$g['DurationHours'];}}
        usort($details,fn($a,$b)=>strcmp((string)$b['CompletionDate'],(string)$a['CompletionDate']));foreach($stats as &$m){$m['AverageHours']=$m['DurationSamples']>0?$m['_sum']/$m['DurationSamples']:0.0;unset($m['_sum']);}unset($m);
        $query=http_build_query(['date_from'=>date('Y-m-d',$start),'date_to'=>date('Y-m-d',$end)]);
        return['DateFromInput'=>date('Y-m-d',$start),'DateToInput'=>date('Y-m-d',$end),'PeriodLabel'=>date('d/m/Y',$start).'–'.date('d/m/Y',$end),'TotalCompleted'=>count($details),'Metrics'=>array_values($stats),'Details'=>$details,'ExportURL'=>'/pelaporan/performa.xlsx?'.$query];
    }
    private function performance(array $events,string $from='',string $to=''):array{$start=$from!==''?strtotime($from.' 00:00:00'):strtotime('first day of january');$end=$to!==''?strtotime($to.' 23:59:59'):time();$counts=['auction_completed'=>0,'destruction_completed'=>0,'grant_completed'=>0,'census_completed'=>0,'pfpd_completed'=>0];foreach($events as $e){$t=strtotime((string)($e['created_at']??''));if($t<$start||$t>$end)continue;$code=$e['code']??'';if(in_array($code,['alokasi_hasil_lelang','laku'],true))$counts['auction_completed']++;if($code==='ba_musnah')$counts['destruction_completed']++;if($code==='ba_serah_terima')$counts['grant_completed']++;if($code==='pencacahan')$counts['census_completed']++;if($code==='penelitian_pfpd')$counts['pfpd_completed']++;}$counts['total_completed']=array_sum($counts);$counts['period_label']=date('d/m/Y',$start).'–'.date('d/m/Y',$end);$counts['from']=date('Y-m-d',$start);$counts['to']=date('Y-m-d',$end);return$counts;}
    private function inventoryGroups(array $items):array{$research=[];$physical=[];foreach($items as $i){$request=trim((string)($i['research_request_no']??''));if($request!==''&&($i['status_code']??'')==='request_penelitian_pfpd'){$research[$request]['RequestNo']=$request;$research[$request]['RequestDate']=$i['research_request_date']??null;$research[$request]['Items'][]=$i;}$key=(string)($i['physical_unit_id']??$i['id']);$physical[$key]['TargetID']=$key;$physical[$key]['LoadType']=$i['load_type']??'';$physical[$key]['ContainerNo']=$i['container_no']??'';$physical[$key]['ContainerSize']=$i['container_size']??'';$physical[$key]['Items'][]=$i;}$physical=array_values($physical);foreach($physical as &$p)$p['SearchValue']=mb_strtolower(($p['ContainerNo']??'').' '.implode(' ',array_column($p['Items'],'description')));unset($p);return['research'=>array_values($research),'physical'=>$physical];}
    private function auctionScheduleGroups(array $processes):array{$groups=[];foreach($processes as $p){$no=trim((string)($p['schedule_document_no']??''));if($no===''||($p['status_code']??'')!=='jadwal_lelang')continue;$groups[$no]['ScheduleNo']=$no;$groups[$no]['ScheduleDate']=$p['schedule_document_date']??null;$groups[$no]['Items'][]=$p;}return array_values($groups);}
    private function allowedInventoryActions(array $session):array{return array_values(array_filter(Domain::INVENTORY_ACTIONS,fn($a)=>Domain::can($session,Domain::actionPermission($a['Code']))));}
    private function inventoryManagementPermissions():array{$p=['inventory.create.btd','inventory.create.bdn','inventory.create.titipan'];foreach(Domain::INVENTORY_ACTIONS as $a)$p[]=Domain::actionPermission($a['Code']);return array_values(array_unique($p));}
    private function processMeta(string $type):array{return match($type){'lelang'=>['lelang','Lelang','lelang','auction.view','auction.manage'],'musnah'=>['musnah','Pemusnahan','pemusnahan','destruction.view','destruction.manage'],'hibah'=>['hibah','Hibah / PSP','hibah/PSP','grant.view','grant.manage'],default=>throw new ApiException('Jenis proses tidak valid.',404)};}
    private function reportData(Request $r):array
    {
        $preset=trim((string)$r->query('preset'));$scope=trim((string)$r->query('scope'));$location=trim((string)$r->query('location'));
        $sort='newest';if($preset==='active_tpp'){$scope='active';$location='tpp';$sort='tpp';}elseif($preset==='overdue_60'){$scope='active';$sort='oldest';}elseif($preset==='auction_ready'){$scope='active';$sort='value_desc';}elseif($preset==='at_tps'){$scope='active';$location='tps';$sort='oldest';}elseif($preset==='bmmn_allocation'){$scope='active';$sort='oldest';}elseif($preset==='completed'){$scope='completed';$sort='newest';}elseif($preset==='btd'){$scope=$scope?:'all';$sort='determination_newest';}
        if(!in_array($scope,['active','all','completed'],true))$scope='active';if(!in_array($location,['tpp','tps'],true))$location='';
        $minAge=max(0,min(36500,(int)$r->query('min_age',($preset==='overdue_60'?60:0))));$minValue=$this->money($r->query('min_value'));$maxValue=$this->money($r->query('max_value'));if($minValue>0&&$maxValue>0&&$minValue>$maxValue)[$minValue,$maxValue]=[$maxValue,$minValue];
        $type=strtoupper(trim((string)$r->query('type')));if($preset==='btd')$type='BTD';if($preset==='bmmn_allocation')$type='BMMN';if($type!==''&&!in_array($type,Domain::TYPES,true))$type='';
        $filter=['allowed_types'=>Domain::allowedTypes($this->session($r)),'type'=>$type,'facility_id'=>(string)$r->query('tpp'),'status'=>(string)$r->query('status'),'query'=>(string)$r->query('q'),'item_kind'=>(string)$r->query('item_kind'),'goods_condition'=>(string)$r->query('goods_condition'),'category'=>(string)$r->query('category'),'allocation_purpose'=>(string)$r->query('allocation_purpose'),'min_value'=>$minValue,'max_value'=>$maxValue,'date_from'=>(string)$r->query('date_from'),'date_to'=>(string)$r->query('date_to'),'preset'=>$preset,'sort'=>$sort,'include_inactive'=>$scope==='all','only_inactive'=>$scope==='completed','location_scope'=>$location,'limit'=>50000];if($scope==='active')$filter['include_inactive']=false;if($minAge>0)$filter['age_before']=date('Y-m-d',strtotime('-'.$minAge.' days'));
        $items=$this->store->listInventory($filter);$titleDescription=$this->reportPresetCopy($preset);$report=$this->reportOptions($r,$preset,$titleDescription[0],$titleDescription[1],['Scope'=>$scope,'Location'=>$location,'ItemKind'=>$filter['item_kind'],'GoodsCondition'=>$filter['goods_condition'],'Category'=>$filter['category'],'AllocationPurpose'=>$filter['allocation_purpose'],'MinValue'=>$minValue?:'','MaxValue'=>$maxValue?:'','MinAge'=>$minAge?:'','DateFrom'=>$filter['date_from'],'DateTo'=>$filter['date_to']]);return[$filter,$items,$report];
    }
    private function reportOptions(Request $r,string $preset,string $title,string $description,array $extra=[]):array{$query=$r->query;unset($query['page'],$query['page_size']);$query['preset']=$preset;if($preset==='')unset($query['preset']);$qs=http_build_query(array_filter($query,fn($v)=>$v!==''&&$v!==null));$suffix=$qs!==''?'?'.$qs:'';return array_merge(['Preset'=>$preset,'Title'=>$title,'Description'=>$description,'ExportURL'=>'/pelaporan.csv'.$suffix,'CSVExportURL'=>'/pelaporan.csv'.$suffix,'ExcelExportURL'=>'/pelaporan.xlsx'.$suffix],$extra);}
    private function reportPresetCopy(string $preset):array{return match($preset){'active_tpp'=>['Barang aktif per TPP','Daftar barang aktif yang saat ini tersebar dan berada di TPP.'],'overdue_60'=>['BTD/BDN 60 hari belum ditindaklanjuti','Barang BTD atau BDN yang telah berumur sekurangnya 60 hari dan masih pada status penetapan awal.'],'auction_ready'=>['Potensi barang siap lelang','Barang bernilai yang sudah diteliti PFPD atau berstatus BMMN, belum masuk proses, diurutkan dari nilai tertinggi.'],'at_tps'=>['Barang aktif masih di TPS','Daftar barang aktif yang belum dipindahkan dari TPS asal ke TPP.'],'bmmn_allocation'=>['BMMN menunggu peruntukan','Daftar BMMN aktif yang belum masuk proses lelang, musnah, atau hibah/PSP.'],'completed'=>['Riwayat barang selesai','Daftar barang yang telah keluar dari inventory aktif.'],'btd'=>['Laporan BTD','Rekap lengkap per dokumen BTD yang memuat BL, manifest, TPS asal, TPP, kontainer/LCL, rincian barang, nilai, dan status.'],default=>['Laporan kustom','Gabungkan rentang tanggal, status inventory, lokasi, nilai, umur, jenis, dan TPP sesuai kebutuhan.']};}
    private function btdRows(array $items):array
    {
        $docs=[];$unique=static function(array $values,string $value):array{$value=trim($value);if($value!==''&&!in_array($value,$values,true))$values[]=$value;return$values;};
        foreach($items as $i){if(($i['item_type']??'')!=='BTD')continue;$key=mb_strtoupper(trim((string)($i['determination_no']??''))).'|'.substr((string)($i['determination_date']??''),0,10);if(!isset($docs[$key]))$docs[$key]=['DeterminationNo'=>(string)($i['determination_no']??''),'DeterminationDate'=>$i['determination_date']??null,'BLNo'=>[],'BLDate'=>[],'ManifestNo'=>[],'ManifestDate'=>[],'ManifestPosition'=>[],'LoadType'=>[],'OriginWarehouse'=>[],'FacilityName'=>[],'LocationStatus'=>[],'ContainerSummary'=>[],'GoodsSummary'=>[],'OwnerName'=>[],'StatusLabel'=>[],'InventoryStatus'=>[],'ContainerUnits'=>[],'ItemCount'=>0,'TotalValue'=>0];$d=&$docs[$key];$d['ItemCount']++;$d['TotalValue']+=(float)($i['goods_value']??0);foreach(['BLNo'=>'bl_no','ManifestNo'=>'manifest_no','ManifestPosition'=>'manifest_position','LoadType'=>'load_type','OriginWarehouse'=>'origin_warehouse','FacilityName'=>'facility_name','LocationStatus'=>'location_status','OwnerName'=>'owner_name','StatusLabel'=>'status_label'] as $out=>$in)$d[$out]=$unique($d[$out],(string)($i[$in]??''));foreach(['BLDate'=>'bl_date','ManifestDate'=>'manifest_date'] as $out=>$in){$v=(string)($i[$in]??'');if($v!=='')$d[$out]=$unique($d[$out],date('d/m/Y',strtotime($v)));}$load=strtoupper((string)($i['load_type']??''));$unit=$load==='FCL'?trim((string)($i['physical_unit_id']??$i['container_no']??'')):'LCL|'.$key;if($unit==='')$unit=(string)($i['id']??uniqid());$d['ContainerUnits'][$unit]=true;$container=$load==='FCL'?trim((string)($i['container_no']??'')).((string)($i['container_size']??'')!==''?' ['.(string)$i['container_size']."]'":''):'LCL'.((float)($i['estimated_volume_m3']??0)>0?' ['.(string)$i['estimated_volume_m3'].' m³]':'');$d['ContainerSummary']=$unique($d['ContainerSummary'],$container);$goodsPrefix=$load==='FCL'&&($i['container_no']??'')!==''?(string)$i['container_no']:'LCL';$parts=array_filter([(string)($i['description']??''),(string)($i['item_kind']??''),(string)($i['goods_condition']??''),trim(rtrim(rtrim(number_format((float)($i['quantity']??0),2,'.',''),'0'),'.').' '.(string)($i['unit']??''))]);$d['GoodsSummary'][]=$goodsPrefix.' ('.implode('; ',$parts).')';$d['InventoryStatus']=$unique($d['InventoryStatus'],!empty($i['is_active'])?'Aktif':'Selesai');unset($d);}
        $rows=[];foreach($docs as $d){$status=count($d['InventoryStatus'])>1?'Campuran':($d['InventoryStatus'][0]??'—');$rows[]=['DeterminationNo'=>$d['DeterminationNo'],'DeterminationDate'=>$d['DeterminationDate'],'BLNo'=>implode('; ',$d['BLNo']),'BLDate'=>implode('; ',$d['BLDate']),'ManifestNo'=>implode('; ',$d['ManifestNo']),'ManifestDate'=>implode('; ',$d['ManifestDate']),'ManifestPosition'=>implode('; ',$d['ManifestPosition']),'LoadType'=>implode('; ',$d['LoadType']),'OriginWarehouse'=>implode('; ',$d['OriginWarehouse']),'FacilityName'=>implode('; ',$d['FacilityName']),'LocationStatus'=>implode('; ',$d['LocationStatus']),'ContainerSummary'=>implode('; ',$d['ContainerSummary']),'ContainerCount'=>count($d['ContainerUnits']),'GoodsSummary'=>implode('; ',$d['GoodsSummary']),'ItemCount'=>$d['ItemCount'],'TotalValue'=>$d['TotalValue'],'OwnerName'=>implode('; ',$d['OwnerName']),'StatusLabel'=>implode('; ',$d['StatusLabel']),'InventoryStatus'=>$status];}
        usort($rows,fn($a,$b)=>strcmp((string)$b['DeterminationDate'],(string)$a['DeterminationDate']));return$rows;
    }

    private function correctionRows(array $corrections): array
    {
        $rows = [];
        foreach ($corrections as $record) {
            $changes = (array) ($record['change_details'] ?? []);
            if ($changes === []) {
                $rows[] = ['Record' => $record, 'Legacy' => true, 'Change' => []];
                continue;
            }
            foreach ($changes as $change) {
                $rows[] = ['Record' => $record, 'Legacy' => false, 'Change' => (array) $change];
            }
        }
        return $rows;
    }

    private function splitReconciliations(array $records): array
    {
        $regular = [];
        $corrections = [];
        foreach ($records as $record) {
            if (($record['reconciliation_type'] ?? '') === 'data_correction') {
                $corrections[] = $record;
            } else {
                $regular[] = $record;
            }
        }
        return [$regular, $corrections];
    }

    private function filterReconciliationsForSession(array $records, array $session): array
    {
        if (($session['Role'] ?? '') === 'admin') {
            return $records;
        }
        $allowed = Domain::allowedTypes($session);
        return array_values(array_filter($records, static function (array $record) use ($allowed): bool {
            $type = strtoupper((string) ($record['inventory_type'] ?? ''));
            return $type !== '' && in_array($type, $allowed, true);
        }));
    }

    private function pagination(Request $r,int $page,int $size,int $total):array{$pages=max(1,(int)ceil($total/$size));$page=min($page,$pages);$build=function(array $changes)use($r){$q=array_merge($r->query,$changes);foreach($q as $k=>$v)if($v===''||$v===null)unset($q[$k]);return$r->path.($q?'?'.http_build_query($q):'');};$sizes=[];foreach([10,20,50,100] as $n)$sizes[]=['Value'=>$n,'Selected'=>$n===$size,'URL'=>$build(['page_size'=>$n,'page'=>1])];return['Page'=>$page,'PageSize'=>$size,'TotalItems'=>$total,'TotalPages'=>$pages,'StartItem'=>$total?($page-1)*$size+1:0,'EndItem'=>min($total,$page*$size),'HasPrevious'=>$page>1,'HasNext'=>$page<$pages,'PreviousURL'=>$build(['page'=>$page-1]),'NextURL'=>$build(['page'=>$page+1]),'Sizes'=>$sizes];}
    private function pageSize(mixed $v):int{$n=(int)$v;return in_array($n,[10,20,50,100],true)?$n:20;}
    private function facilityName(array $facilities,string $id):string{foreach($facilities as $f)if(($f['id']??'')===$id)return(string)$f['name'];return'Gabungan seluruh TPP';}
    private function values(mixed $v):array{if(is_array($v))return array_values(array_unique(array_filter(array_map(fn($x)=>trim((string)$x),$v))));$s=trim((string)$v);if($s==='')return[];return array_values(array_unique(array_filter(array_map('trim',preg_split('/[,\r\n]+/',$s)?:[]))));}
    private function jsonArray(mixed $v):array{$a=json_decode((string)$v,true);return is_array($a)?(array_is_list($a)?$a:[$a]):[];}
    private function jsonObject(mixed $v):array{$a=json_decode((string)$v,true);return is_array($a)?$a:[];}
    private function formMap(Request $r,array $keys):array{$out=[];foreach($keys as $key)$out[$key]=$r->input($key);return$out;}
    private function bool(mixed $v):bool{return filter_var($v,FILTER_VALIDATE_BOOL)||in_array(strtolower((string)$v),['1','ya','yes','on','sudah'],true);}
    private function number(mixed $v):float{$s=str_replace(',','.',preg_replace('/[^0-9,.-]/','',(string)$v));return is_numeric($s)?(float)$s:0;}
    private function money(mixed $v):int{return(int)preg_replace('/\D/','',(string)$v);}
    private function normalizeHeader(string $v):string{$v=mb_strtolower(trim($v));$map=['nomor penetapan'=>'determination_no','no penetapan'=>'determination_no','tanggal penetapan'=>'determination_date','nomor dokumen'=>'determination_no','tanggal dokumen'=>'determination_date','nomor bl'=>'bl_no','tanggal bl'=>'bl_date','nomor manifest'=>'manifest_no','tanggal manifest'=>'manifest_date','pos manifest'=>'manifest_position','kategori bdn'=>'category','kategori titipan'=>'entrusted_category','kantor penitip'=>'source_office','uraian barang'=>'description','jenis barang'=>'item_kind','nilai barang'=>'goods_value','jumlah barang'=>'quantity','detail jumlah'=>'quantity_detail','satuan'=>'unit','kondisi barang'=>'goods_condition','nama pemilik'=>'owner_name','alamat pemilik'=>'owner_address','tps asal'=>'origin_warehouse','tpp'=>'facility_id','lokasi'=>'location','jenis muatan'=>'load_type','nomor kontainer'=>'container_no','ukuran kontainer'=>'container_size','volume m3'=>'estimated_volume_m3'];$v=preg_replace('/\s+/',' ',$v);return$map[$v]??str_replace(' ','_',$v);}
    private function mapImportRow(array $row,string $type,int $line):array{$row['type']=$type;$row['reference_no']=trim((string)($row['reference_no']??''));$row['quantity']=$this->number($row['quantity']??0);$row['goods_value']=$this->money($row['goods_value']??0);$row['estimated_volume_m3']=$this->number($row['estimated_volume_m3']??0);$row['load_type']=strtoupper(trim((string)($row['load_type']??(!empty($row['container_no'])?'FCL':'LCL'))));$row['at_tpp']=trim((string)($row['facility_id']??''))!=='';$row['occupancy_primary']=true;if(trim((string)($row['determination_no']??''))===''||trim((string)($row['description']??''))===''||$row['quantity']<=0)throw new ApiException('Baris '.$line.': nomor dokumen, uraian, dan jumlah wajib diisi.',422);return$row;}
    private function optionalDocument(Request $r):string{$file=$r->files['document_file']??null;if(!is_array($file)||($file['error']??UPLOAD_ERR_NO_FILE)===UPLOAD_ERR_NO_FILE)return'';if(($file['error']??UPLOAD_ERR_OK)!==UPLOAD_ERR_OK)throw new ApiException('Upload dokumen gagal.',422);$doc=$this->store->createDocument($file,$this->actor($r));return(string)($doc['id']??'');}
    private function safeReturn(Request $r,string $fallback):string{$target=(string)$r->input('return_to',$r->header('referer'));if($target==='')return$fallback;$path=parse_url($target,PHP_URL_PATH)?:'';$query=parse_url($target,PHP_URL_QUERY);if(!str_starts_with($path,'/')||str_starts_with($path,'//'))return$fallback;return$path.($query?'?'.$query:'');}
    private function back(Request $r,string $message,string $fallback='/'):Response{$url=$this->safeReturn($r,$fallback);$separator=str_contains($url,'?')?'&':'?';return Response::redirect($url.$separator.'success='.rawurlencode($message));}
    private function errorResponse(Request $r,string $message,int $status):Response{if($r->acceptsJson())return Response::json(['error'=>$message],$status);if(in_array($r->method,['POST','PUT','PATCH','DELETE'],true)){return Response::redirect($this->safeReturn($r,'/').(str_contains($this->safeReturn($r,'/'),'?')?'&':'?').'error='.rawurlencode($message));}return Response::html('<!doctype html><html lang="id"><meta charset="utf-8"><title>Kesalahan</title><body><h1>Kesalahan</h1><p>'.htmlspecialchars($message,ENT_QUOTES|ENT_SUBSTITUTE,'UTF-8').'</p><p><a href="/">Kembali</a></p></body></html>',$status);}
    private function httpStatus(ApiException $e):int{$code=$e->getCode();return$code>=400&&$code<=599?$code:500;}
    private function logException(\Throwable $e,Request $r):void{$line=gmdate('c').' '.$r->method.' '.$r->path.' '.$e::class.': '.$e->getMessage()."\n".$e->getTraceAsString()."\n\n";@file_put_contents($this->basePath.'/storage/logs/app.log',$line,FILE_APPEND|LOCK_EX);}
    private function htmlTable(array $headers,array $rows):string{$out='<html><head><meta charset="utf-8"></head><body><table border="1"><thead><tr>';foreach($headers as $h)$out.='<th>'.htmlspecialchars($h).'</th>';$out.='</tr></thead><tbody>';foreach($rows as $row){$out.='<tr>';foreach($row as $v)$out.='<td>'.htmlspecialchars((string)$v).'</td>';$out.='</tr>';}$out.='</tbody></table></body></html>';return$out;}
}
