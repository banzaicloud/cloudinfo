import { Directive, ElementRef, HostBinding, Input } from '@angular/core';

@Directive({
  selector: '[appBorderDetailsStatus]',
})
export class BanzaiTableBorderDetailsStatusDirective {

  private _isOpen: boolean;
  private lastClass: string; // this is necessary because of updating

  @Input() set isOpen(value: boolean) {
    this._isOpen = value;
    this.updateClasses();
  }

  @HostBinding('class') hostClass;

  constructor(
    private el: ElementRef,
  ) { }

  get isOpen(): boolean {
    return this._isOpen;
  }

  private getStyle(): string {
    let baseClass = 'table-detail-row';
    if (this.isOpen) {
      baseClass = baseClass.concat(' open');
    } else {
      baseClass = baseClass.concat(' closed');
    }
    return baseClass;
  }

  private updateClasses() {
    let currentClasses = this.el.nativeElement.className.toString();
    const statusClass = this.getStyle();
    if (this.lastClass) {
      currentClasses = currentClasses.replace(this.lastClass, '');
    }
    this.lastClass = statusClass;
    this.hostClass = `${statusClass} ${currentClasses}`;
  }

}
