<?php
declare(strict_types=1);
namespace Livira\Support;

final class View
{
    public function __construct(private readonly string $viewsPath) {}

    public function render(string $view, array $data): string
    {
        $data = $this->defaults($data);
        $root = $ctx = $data;
        $viewFile = $this->viewsPath.'/'.$view.'.php';
        $layoutFile = $this->viewsPath.'/layout.php';
        if (!is_file($viewFile) || !is_file($layoutFile)) throw new \RuntimeException('View tidak ditemukan: '.$view);
        ob_start();
        include $viewFile;
        $content = (string)ob_get_clean();
        $root = $ctx = $data;
        ob_start();
        include $layoutFile;
        return (string)ob_get_clean();
    }

    private function defaults(array $d): array
    {
        $defaults = [
            'Title'=>'LIVIRA','Subtitle'=>'','Active'=>'','AuthPage'=>false,'SignupPage'=>false,'OTPPage'=>false,
            'ForgotPasswordPage'=>false,'ResetPasswordPage'=>false,'VerifyEmail'=>'','CaptchaToken'=>'','DemoMode'=>false,
            'User'=>[],'CSRF'=>'','Success'=>'','Error'=>'','Facilities'=>[],'Stats'=>[],'DashboardRows'=>[],
            'DashboardOccupancy'=>[],'DashboardScope'=>'','DashboardInventoryScope'=>'all_office','DashboardInventoryLabel'=>'Seluruh cakupan kantor Tanjung Priok',
            'Items'=>[],'EligibleItems'=>[],'Processes'=>[],'CandidateProcesses'=>[],'InventoryActions'=>[],'ProcessActions'=>[],
            'ProcessType'=>'','ProcessTitle'=>'','ProcessSingular'=>'','ProcessDashboard'=>[],'AuctionDashboard'=>[],'DestructionDashboard'=>[],
            'GrantDashboard'=>[],'ProcessModals'=>[],'Query'=>'','FacilityID'=>'','InventoryType'=>'','Status'=>'','Sort'=>'newest','History'=>false,
            'SearchPerformed'=>false,'Search'=>[],'Now'=>date(DATE_ATOM),'ActiveProcesses'=>0,'ClosedProcesses'=>0,'ReportTotal'=>0,
            'ReportActive'=>0,'ReportClosed'=>0,'ReportTotalValue'=>0,'ReportAtTPP'=>0,'ReportTransactionTotal'=>0,'Report'=>[],
            'TPSNames'=>[],'BDNCategoryNames'=>[],'ItemKindNames'=>[],'GoodsConditionNames'=>[],'AllocationPurposeNames'=>[],'UnitNames'=>[],
            'LoadTypeOptions'=>[],'ContainerSizeOptions'=>[],'ExitOptions'=>[],'TransferTypeOptions'=>[],'Users'=>[],'Roles'=>[],
            'PermissionDefinitions'=>[],'Parameters'=>[],'AdminSection'=>'','PendingUsers'=>0,'VerifiedPendingUsers'=>0,'CanManage'=>false,
            'CanCreateInventory'=>false,'CanCreateBTD'=>false,'CanCreateBDN'=>false,'CanCreateTitipan'=>false,'CanRunInventoryActions'=>false,
            'CanEditCapacity'=>false,'IdleTimeoutSeconds'=>1800,'Notifications'=>[],'NotificationCount'=>0,
            'Pagination'=>['Page'=>1,'PageSize'=>20,'TotalItems'=>0,'TotalPages'=>1,'StartItem'=>0,'EndItem'=>0,'HasPrevious'=>false,'HasNext'=>false,'Sizes'=>[]],
            'ResearchRequestGroups'=>[],'CensusTargetGroups'=>[],'RelocationTargetGroups'=>[],'AuctionScheduleGroups'=>[],
            'Reconciliations'=>[],'DataCorrections'=>[],'DataCorrectionRows'=>[],'ReconciliationTab'=>'rekonsiliasi','EntrustedCategoryNames'=>[],
            'ReportReconciliation'=>false,'ReportDataCorrection'=>false,'ReportBTD'=>false,'BTDReportRows'=>[],'ReportPerformance'=>false,
            'Performance'=>[],'PerformanceOpen'=>false,
        ];
        return array_replace_recursive($defaults,$d);
    }
}
