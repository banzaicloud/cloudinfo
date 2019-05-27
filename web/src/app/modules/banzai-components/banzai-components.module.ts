import { NgModule } from '@angular/core';
import { CommonModule, DatePipe, DecimalPipe } from '@angular/common';
import { BanzaiSelectorComponent } from './components/banzai-selector/banzai-selector.component';
import {
  MatProgressSpinnerModule,
  MatRippleModule,
  MatSelectModule,
  MatSortModule,
  MatTableModule,
  MatTooltipModule,
} from '@angular/material';
import { BanzaiTableComponent } from './components/banzai-table/banzai-table.component';
import { BanzaiTableCellTextOverflowDirective } from './components/banzai-table/directives/banzai-table-cell-text-overflow.directive';
import { BanzaiTableBorderDetailsStatusDirective } from './components/banzai-table/directives/banzai-table-border-details-status.directive';
import { BanzaiTableRowStyleDirective } from './components/banzai-table/directives/banzai-table-row-style.directive';
import { BanzaiTableStyleDirective } from './components/banzai-table/directives/banzai-table-style.directive';
import { BanzaiTableCellConfigDirective } from './components/banzai-table/directives/banzai-table-cell-config.directive';
import { BanzaiTableCellWidthDirective } from './components/banzai-table/directives/banzai-table-cell-width.directive';
import { TruncateAtMiddlePipe } from './components/banzai-table/pipe/truncate-at-middle.pipe';
import { TimeAgoPipe } from 'time-ago-pipe';
import { BanzaiCopyIconComponent } from './components/banzai-copy-icon/banzai-copy-icon.component';
import { ClipboardModule } from 'ngx-clipboard';
import { ToFixedNumberPipe } from './components/banzai-table/pipe/to-fixed-number.pipe';

@NgModule({
  imports: [
    CommonModule,
    MatSelectModule,
    MatTableModule,
    MatSortModule,
    MatTooltipModule,
    MatRippleModule,
    ClipboardModule,
    MatProgressSpinnerModule,
  ],
  declarations: [
    BanzaiSelectorComponent,
    BanzaiTableComponent,
    TimeAgoPipe,
    TruncateAtMiddlePipe,
    BanzaiTableCellWidthDirective,
    BanzaiTableCellConfigDirective,
    BanzaiTableStyleDirective,
    BanzaiTableRowStyleDirective,
    BanzaiTableCellTextOverflowDirective,
    BanzaiTableBorderDetailsStatusDirective,
    BanzaiCopyIconComponent,
    ToFixedNumberPipe,
  ],
  exports: [
    BanzaiSelectorComponent,
    BanzaiTableComponent,
    BanzaiCopyIconComponent,
  ],
  providers: [
    TruncateAtMiddlePipe,
    ToFixedNumberPipe,
    DatePipe,
    DecimalPipe,
  ],
})
export class BanzaiComponentsModule {
}
