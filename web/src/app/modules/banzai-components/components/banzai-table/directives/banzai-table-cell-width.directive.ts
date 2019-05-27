import { Directive, HostBinding, Input, OnInit } from '@angular/core';

@Directive({
  selector: '[appCellWidth]',
})
export class BanzaiTableCellWidthDirective implements OnInit {

  @Input() customWidth?: string;

  @HostBinding('style.width') width;

  constructor() { }

  ngOnInit(): void {
    if (this.customWidth) {
      this.width = this.customWidth;
    }
  }

}
