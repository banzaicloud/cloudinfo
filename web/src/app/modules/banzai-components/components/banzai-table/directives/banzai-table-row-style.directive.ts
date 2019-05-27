import { Directive, ElementRef, HostBinding, Input, OnInit } from '@angular/core';

import { TableRowDesignConfig, TableRowMarkDesign } from '../model/tabledata';

@Directive({
  selector: '[appRowStyle]',
})
export class BanzaiTableRowStyleDirective implements OnInit {

  @Input() rowDesignConfig: TableRowDesignConfig;

  @Input() set selected(value: boolean) {
    this._selected = value;
    this.updateClass();
  }

  @HostBinding('class') hostClass;

  private _selected: boolean;

  constructor(
    private el: ElementRef,
  ) { }

  ngOnInit(): void {
    this.updateClass();
  }

  private updateClass() {
    let currentClasses = this.el.nativeElement.className.toString();
    const markClass = this.getRowMarkClass();

    if (this.selected) {
      currentClasses = currentClasses.concat(' ').concat(markClass);
    } else {
      currentClasses = currentClasses.replace(markClass, '');
    }

    this.hostClass = `table-element-row ${currentClasses}`;
  }

  get selected(): boolean {
    return this._selected;
  }

  private getRowMarkClass(): string {
    if (this.rowDesignConfig) {

      switch (this.rowDesignConfig.mark) {
        case TableRowMarkDesign.LightRed:
          return 'banzai-table-row-mark-light-red';
        case TableRowMarkDesign.Smoke:
          return 'banzai-table-row-mark-smoke';
        default:
          return '';
      }

    }

    return '';
  }


}
