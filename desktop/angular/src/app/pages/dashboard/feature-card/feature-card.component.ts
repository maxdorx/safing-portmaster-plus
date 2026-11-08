import { coerceBooleanProperty } from '@angular/cdk/coercion';
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnDestroy, SimpleChanges, inject } from '@angular/core';
import { Router } from '@angular/router';
import { BoolSetting, ConfigService, Feature } from '@safing/portmaster-api';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-feature-card',
  templateUrl: './feature-card.component.html',
  styleUrls: ['./feature-card.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class FeatureCardComponent implements OnChanges, OnDestroy {
  private readonly cdr = inject(ChangeDetectorRef);
  private readonly configService = inject(ConfigService);
  private readonly router = inject(Router);

  private configValueSubscription = Subscription.EMPTY;

  @Input()
  set disabled(v: any) {
    this._disabled = coerceBooleanProperty(v)
  }
  get disabled() { return this._disabled }
  _disabled = false;

  get comingSoon() { return this.feature?.ComingSoon || false }

  @Input()
  feature?: Feature;

  configValue: boolean | undefined = undefined;

  ngOnChanges(changes: SimpleChanges): void {
    if ('feature' in changes) {
      this.configValueSubscription.unsubscribe();
      this.configValueSubscription = Subscription.EMPTY;

      if (!!this.feature?.ConfigKey) {
        this.configValueSubscription =
          this.configService.watch<BoolSetting>(this.feature!.ConfigKey)
            .subscribe(value => {
              this.configValue = value;
              this.cdr.markForCheck();
            });
      }

    }
  }

  ngOnDestroy() {
    this.configValueSubscription.unsubscribe();
  }

  updateSettingsValue(newValue: boolean) {
    this.configService.save(this.feature!.ConfigKey, newValue)
      .subscribe()
  }

  navigateToConfigScope() {
    if (this.disabled) {
      return;
    }

    let key: string | undefined;
    if (this.feature?.ConfigKey) {
      key = this.feature?.ConfigKey;
    } else {
      key = 'config:' + this.feature?.ConfigScope;
    }

    if (!key) {
      return
    }

    this.router.navigate(['/settings'], {
      queryParams: {
        setting: key,
      }
    })
  }
}
