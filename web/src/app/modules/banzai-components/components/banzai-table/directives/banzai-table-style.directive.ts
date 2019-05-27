import { Directive, ElementRef, HostBinding, Input, OnInit } from '@angular/core';

import { TableDesignConfig, TableDesignType, TableRowHoverDesign } from '../model/tabledata';

@Directive({
  selector: '[appTableStyle]',
})
export class BanzaiTableStyleDirective implements OnInit {

  @Input() designConfig: TableDesignConfig;

  @HostBinding('class') hostClass;

  constructor(
    private el: ElementRef,
  ) { }

  ngOnInit(): void {
    const styleTypeClass = this.getStyleTypeClass();
    const hoverClass = this.getRowHoverClass();
    const classes = styleTypeClass.concat(' ').concat(hoverClass);

    this.hostClass = `${classes} ${this.el.nativeElement.className.toString()}`;
  }

  private getStyleTypeClass(): string {
    if (this.designConfig) {
      switch (this.designConfig.type) {
        case TableDesignType.Card:
          return 'banzai-table-card';
        case TableDesignType.Border:
          return 'banzai-table-border-rows';
        case TableDesignType.Base:
          return 'banzai-table-base-rows';
        case TableDesignType.BaseWithCheckbox:
          return 'banzai-table-base-rows-with-checkbox';
        case TableDesignType.CardWithTemplateFirst:
          return 'banzai-table-card-rows-with-template-first';
      }
    }

    return '';
  }

  private getRowHoverClass() {
    if (this.designConfig) {

      switch (this.designConfig.rowDesign.hover) {
        case TableRowHoverDesign.White:
          return 'banzai-table-row-white-hover';
      }

    }

    return '';
  }
}
