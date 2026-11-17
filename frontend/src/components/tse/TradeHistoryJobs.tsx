import { useState } from 'react';
import { Loader2, XCircle } from 'lucide-react';
import { Badge } from '../ui/Badge';
import { Button } from '../ui/Button';
import { confirmDialog } from '../ui/dialogs';
import { errorMessage, formatDateTime } from './format';
import { tradeHistoryService, tradeJobStatusLabel, type TradeHistoryJob, type TradeJobStatus } from '../../services/tradeHistoryService';

const statusVariant: Record<TradeJobStatus, 'default' | 'success' | 'danger' | 'warning' | 'primary'> = {
  PENDING: 'warning',
  RUNNING: 'primary',
  DONE: 'success',
  FAILED: 'danger',
  CANCELED: 'default',
};

const fa = (value: number) => value.toLocaleString('fa-IR');

function JobProgress({ job }: { job: TradeHistoryJob }) {
  const percent = job.planned_days > 0 ? Math.min(100, Math.round((100 * job.processed_days) / job.planned_days)) : 0;
  return (
    <div className="space-y-1">
      {job.status === 'RUNNING' && (
        <div className="h-1.5 overflow-hidden rounded-pill bg-surface-muted">
          <div className="h-full rounded-pill bg-action transition-all" style={{ width: `${percent}%` }} />
        </div>
      )}
      <p className="text-caption text-text-subtle">
        {job.total_markets > 0 && <>نماد {fa(job.done_markets)} از {fa(job.total_markets)} · </>}
        روز {fa(job.processed_days)} از {fa(job.planned_days)}
        {job.processed_days > 0 && (
          <>
            {' '}· <span className="text-success">{fa(job.fetched_days)} دریافت</span>
            {' '}· <span className="text-warning-text">{fa(job.empty_days)} خالی</span>
            {' '}· <span className="text-error">{fa(job.failed_days)} خطا</span>
          </>
        )}
        {job.status === 'RUNNING' && job.current_market && <> · در حال دریافت {job.current_market}</>}
      </p>
    </div>
  );
}

/** Lists trade history jobs with their progress; queued and running jobs can be canceled. */
export function TradeHistoryJobs({
  jobs,
  onChanged,
  onError,
  emptyText = 'کاری در صف نیست.',
}: {
  jobs: TradeHistoryJob[];
  onChanged: () => void;
  onError: (message: string) => void;
  emptyText?: string;
}) {
  const [canceling, setCanceling] = useState<number | null>(null);

  const cancel = async (job: TradeHistoryJob) => {
    const confirmed = await confirmDialog({
      title: 'لغو کار',
      message: `کار «${job.title}» لغو شود؟`,
      warning: job.status === 'RUNNING' ? 'روزهایی که تا این لحظه دریافت شده‌اند حفظ می‌شوند.' : undefined,
      confirmLabel: 'لغو کار',
      tone: 'danger',
    });
    if (!confirmed) return;
    setCanceling(job.id);
    try {
      await tradeHistoryService.cancelJob(job.id);
      onChanged();
    } catch (e) {
      onError(errorMessage(e, 'خطا در لغو کار'));
    } finally {
      setCanceling(null);
    }
  };

  if (jobs.length === 0) return <p className="text-body text-text-subtle">{emptyText}</p>;

  return (
    <div className="divide-y divide-border">
      {jobs.map((job) => {
        const active = job.status === 'PENDING' || job.status === 'RUNNING';
        return (
          <div key={job.id} className="flex flex-wrap items-start justify-between gap-3 py-2.5">
            <div className="min-w-0 flex-1 space-y-1">
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant={statusVariant[job.status]}>
                  {job.status === 'RUNNING' && <Loader2 className="me-1 h-3 w-3 animate-spin" />}
                  {job.cancel_requested && active ? 'در حال لغو' : tradeJobStatusLabel[job.status]}
                </Badge>
                <span className="text-body font-medium text-text">{job.title}</span>
                <span className="text-[11px] text-text-subtle">#{fa(job.id)} · {formatDateTime(job.created_at)}</span>
              </div>
              {(job.status !== 'PENDING' || job.processed_days > 0) && <JobProgress job={job} />}
              {job.error && <p className="text-caption text-error">{job.error}</p>}
              {job.finished_at && <p className="text-[11px] text-text-subtle">پایان: {formatDateTime(job.finished_at)}</p>}
            </div>
            {active && !job.cancel_requested && (
              <Button size="sm" variant="ghost" isLoading={canceling === job.id} onClick={() => cancel(job)}>
                <XCircle className="h-4 w-4" />
                لغو
              </Button>
            )}
          </div>
        );
      })}
    </div>
  );
}
