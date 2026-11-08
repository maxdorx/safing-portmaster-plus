import { ChangeDetectionStrategy, ChangeDetectorRef, Component, DestroyRef, OnInit, inject } from "@angular/core";
import { takeUntilDestroyed } from "@angular/core/rxjs-interop";
import { BoolSetting, ConfigService } from "@safing/portmaster-api";
import { fadeInAnimation, fadeOutAnimation } from "../animations";

@Component({
  selector: 'app-feature-scout',
  templateUrl: './feature-scout.html',
  styleUrls: [
    './feature-scout.scss'
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  animations: [
    fadeInAnimation,
    fadeOutAnimation,
  ]
})
export class FeatureScoutComponent implements OnInit {
  private destroyRef = inject(DestroyRef);

  /** Whether or not the Network History is currently enabled */
  historyEnabled = false;

  constructor(
    private configService: ConfigService,
    private cdr: ChangeDetectorRef,
  ) { }

  ngOnInit(): void {
    this.configService.watch<BoolSetting>("history/enable")
    .pipe(takeUntilDestroyed(this.destroyRef))
    .subscribe(value => {
      this.historyEnabled = value;

      this.cdr.markForCheck();
    });
  }

  setHistoryEnabled(v: boolean) {
    this.configService.save(`history/enable`, v)
      .subscribe();
  }
}
