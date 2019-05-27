import { Directive, ElementRef, HostBinding, Input, OnInit } from '@angular/core';

import { CellAlignConfig, BorderConfig, CellPaddingType, CellPositionType } from '../model/tabledata';

@Directive({
  selector: '[appCellConfig]',
})
export class BanzaiTableCellConfigDirective implements OnInit {

  // align config
  @Input() cellAlignConfig?: CellAlignConfig;
  @Input() isHeader?: boolean;

  // border config
  @Input() borderConfig?: BorderConfig;

  // padding config
  @Input() paddingType?: CellPaddingType;

  // position config
  @Input() customPosition?: CellPositionType;

  @HostBinding('class') hostClass;

  constructor(
    private el: ElementRef,
  ) { }

  ngOnInit(): void {

    let classes = this.getAlignClass().concat(' ');
    classes = classes.concat(this.getBorderClass()).concat(' ');
    classes = classes.concat(this.getPaddingClass()).concat(' ');
    classes = classes.concat(this.getPositionClass());

    this.hostClass = `${classes} ${this.el.nativeElement.className}`;
  }

  private getAlignClass(): string {
    if (this.cellAlignConfig) {
      switch (this.cellAlignConfig) {
        case CellAlignConfig.Center:
          return this.isHeader ? 'center' : 'text-center';
      }
    }

    return '';

  }

  private getBorderClass(): string {
    return this.borderConfig ? `table-cell-${this.borderConfig}-border` : '';
  }

  private getPaddingClass(): string {
    return this.paddingType ? `table-cell-padding-${this.paddingType}` : '';
  }

  private getPositionClass(): string {
    return this.paddingType ? `table-cell-position-${this.customPosition}` : '';
  }

}
