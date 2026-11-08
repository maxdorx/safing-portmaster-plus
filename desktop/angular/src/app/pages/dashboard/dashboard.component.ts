import { KeyValue } from "@angular/common";
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, DestroyRef, OnInit, TrackByFunction, inject } from "@angular/core";
import { takeUntilDestroyed } from "@angular/core/rxjs-interop";
import { BandwidthChartResult, ChartResult, Database, FeatureID, Netquery, SPNService, Verdict } from "@safing/portmaster-api";
import { map, repeat } from "rxjs";
import { DefaultBandwidthChartConfig } from "src/app/shared/netquery/line-chart/line-chart";
import { CircularBarChartConfig, splitQueryResult } from "src/app/shared/netquery/circular-bar-chart/circular-bar-chart.component";
import { BytesPipe } from "src/app/shared/pipes/bytes.pipe";

interface BlockedProfile {
  profileID: string;
  count: number;
}

interface BandwidthBarData {
  profile: string;
  profile_name: string;
  series: 'sent' | 'received';
  value: number;
  sent: number;
  received: number;
}

@Component({
  selector: 'app-dashboard',
  changeDetection: ChangeDetectionStrategy.OnPush,
  styleUrls: ['./dashboard.component.scss'],
  templateUrl: './dashboard.component.html',
})
export class DashboardPageComponent implements OnInit {

  private readonly destroyRef = inject(DestroyRef);
  private readonly netquery = inject(Netquery);
  private readonly cdr = inject(ChangeDetectorRef);

  blockedProfiles: BlockedProfile[] = []

  connectionsPerCountry: {
    [country: string]: number
  } = {};

  bandwidthLineChart: BandwidthChartResult<any>[] = [];

  bandwidthBarData: BandwidthBarData[] = [];

  readonly bandwidthBarConfig: CircularBarChartConfig<BandwidthBarData> = {
    stack: 'profile_name',
    seriesKey: 'series',
    seriesLabel: d => {
      if (d === 'sent') {
        return 'Bytes Sent'
      }
      return 'Bytes Received'
    },
    value: 'value',
    ticks: 3,
    colorAsClass: true,
    series: {
      'sent': {
        color: 'text-deepPurple-500 text-opacity-50',
      },
      'received': {
        color: 'text-cyan-800 text-opacity-50',
      }
    },
    formatTick: (tick: number) => {
      return new BytesPipe().transform(tick, '1.0-0')
    },
    formatValue: (stack, series, value, data) => {
      const bytes = new BytesPipe().transform
      return `${stack}\nSent: ${bytes(data?.sent)}\nReceived: ${bytes(data?.received)}`
    },
    formatStack: (sel, data) => {
      const bytes = new BytesPipe().transform

      return sel
        .call(sel => {
          sel.append("text")
            .attr("dy", "0")
            .attr("y", "0")
            .text(d => d)
        })
        .call(sel => {
          sel.append("text")
            .attr("y", 0)
            .attr("dy", "0.8rem")
            .style("font-size", "0.6rem")
            .text(d => {
              const first = data.find(result => result.profile_name === d);
              return `${bytes(first?.sent)} / ${bytes(first?.received)}`
            })
        })
    }
  }

  bwChartConfig = DefaultBandwidthChartConfig;

  activeConnections: number = 0;
  blockedConnections: number = 0;
  activeProfiles: number = 0;
  dataIncoming = 0;
  dataOutgoing = 0;
  connectionChart: ChartResult[] = [];

  countriesPerProfile: { [profile: string]: string[] } = {}

  featureBw = true;

  features$ = inject(SPNService).loadFeaturePackages().pipe(
    map(features => features.filter(feature => feature.ID !== 'spn' && feature.RequiredFeatureID !== FeatureID.SPN && !feature.ComingSoon).map(feature => ({ ...feature, enabled: true })))
  );

  trackCountry: TrackByFunction<KeyValue<string, any>> = (_, ctr) => ctr.key;
  trackApp: TrackByFunction<BlockedProfile> = (_, bp) => bp.profileID;

  async ngOnInit() {
    this.netquery
      .batch({
        bwBarChart: {
          query: {
            internal: { $eq: false },
          },
          select: [
            'profile',
            'profile_name',
            {
              $sum: {
                field: 'bytes_sent',
                as: 'sent'
              }
            },
            {
              $sum: {
                field: 'bytes_received',
                as: 'received'
              }
            },
          ],
          groupBy: ['profile', 'profile_name'],
        },

        profileCount: {
          select: [
            'profile',
            {
              $count: {
                field: '*',
                as: 'totalCount'
              }
            }
          ],
          query: {
            verdict: { $in: [Verdict.Block, Verdict.Drop] }
          },
          groupBy: ['profile'],
          databases: [Database.Live]
        },

        countryStats: {
          select: [
            'country',
            { $count: { field: '*', as: 'totalCount' } },
            { $sum: { field: 'bytes_sent', as: 'bwout' } },
            { $sum: { field: 'bytes_received', as: 'bwin' } },
          ],
          query: {
            allowed: { $eq: true },
          },
          groupBy: ['country'],
          databases: [Database.Live]
        },

        perCountryConns: {
          select: ['profile', 'country', 'active', { $count: { field: '*', as: 'totalCount' } }],
          query: {
            allowed: { $eq: true },
          },
          groupBy: ['profile', 'country', 'active'],
          databases: [Database.Live],
        },

      })
      .pipe(
        repeat({ delay: 10000 }),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe(response => {
        // bandwidth bar chart
        if (response?.bwBarChart){
          const barChartData = response.bwBarChart
            .filter(value => (value.sent + value.received) > 0)
            .sort((a, b) => (b.sent + b.received) - (a.sent + a.received))
            .slice(0, 10);
          this.bandwidthBarData = splitQueryResult(barChartData, ['sent', 'received']) as BandwidthBarData[]
        }

        // profileCount
        this.blockedConnections = 0;
        this.blockedProfiles = [];

        response.profileCount?.forEach(row => {
          this.blockedConnections += row.totalCount;
          this.blockedProfiles.push({
            profileID: row.profile!,
            count: row.totalCount
          })
        });

        // countryStats
        this.connectionsPerCountry = {};
        this.dataIncoming = 0;
        this.dataOutgoing = 0;

        response.countryStats?.forEach(row => {
          this.dataIncoming += row.bwin;
          this.dataOutgoing += row.bwout;

          if (row.country === '') {
            return
          }

          this.connectionsPerCountry[row.country!] = row.totalCount || 0;
        })

        // perCountryConns
        let profiles = new Set<string>();

        this.activeConnections = 0;
        this.countriesPerProfile = {};

        response.perCountryConns?.forEach(row => {
          profiles.add(row.profile!);

          if (row.active) {
            this.activeConnections += row.totalCount;
          }

          const arr = (this.countriesPerProfile[row.profile!] || []);
          arr.push(row.country!)
          this.countriesPerProfile[row.profile!] = arr;
        });

        this.activeProfiles = profiles.size;

        this.cdr.markForCheck();
      })

    // Charts

    this.netquery
      .activeConnectionChart({})
      .pipe(
        repeat({ delay: 10000 }),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe(result => {
        this.connectionChart = result;
        this.cdr.markForCheck();
      })

    this.netquery
      .bandwidthChart({}, undefined, 60)
      .pipe(
        repeat({ delay: 10000 }),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe(bw => {
        this.bandwidthLineChart = bw;
        this.cdr.markForCheck();
      })

  }
}
