import { Directive, ElementRef, HostBinding, Input, OnInit } from '@angular/core';

@Directive({
  selector: '[appCellTextOverflow]',
})
export class BanzaiTableCellTextOverflowDirective implements OnInit {

  @Input() enabledOverflow?: boolean;

  @HostBinding('class') hostClass;

  constructor(
    private el: ElementRef,
  ) { }

  ngOnInit(): void {
    this.hostClass = `${this.getOverflowClass()} ${this.el.nativeElement.className}`;
  }

  private getOverflowClass(): string {
    if (this.enabledOverflow) {
      return 'cell-text-overflow';
    }
    return '';
  }

}
