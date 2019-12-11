import { Component, ElementRef, EventEmitter, OnDestroy, OnInit, Output, ViewChild } from '@angular/core';
import { Subject } from 'rxjs';
import { ActivatedRoute, Router } from '@angular/router';
import { debounceTime, distinctUntilChanged, pluck, takeUntil } from 'rxjs/operators';

@Component({
  selector: 'app-search',
  templateUrl: './search.component.html',
  styleUrls: ['./search.component.scss'],
})
export class SearchComponent implements OnInit, OnDestroy {

  @Output() filterChanged: EventEmitter<string> = new EventEmitter<string>();

  @ViewChild('filterInput', { static: true }) filterInput: ElementRef;

  private $unsubscribe = new Subject();
  private filter$ = new Subject<string>();
  public filter: string;

  constructor(
    private activatedRouter: ActivatedRoute,
    private router: Router,
  ) {}

  ngOnInit() {
    this.listenOnFilterChange();
    this.listenOnQueryParams();
  }

  private listenOnQueryParams() {
    this.activatedRouter.queryParams
      .pipe(
        pluck('filter'),
        distinctUntilChanged(),
        takeUntil(this.$unsubscribe))
      .subscribe((filter: string = '') => {
        this.filter = filter;
        this.applyFilter(filter);

        if (this.filter) {
          this.filterInput.nativeElement.focus();
        }
      });
  }

  public applyFilter(filter: string = this.filter) {
    this.filter$.next(filter);
  }

  private listenOnFilterChange() {
    this.filter$
      .pipe(
        distinctUntilChanged(),
        debounceTime(50),
        takeUntil(this.$unsubscribe))
      .subscribe((filter: string) => {
        this.router.navigate(['.'], {
          relativeTo: this.activatedRouter,
          queryParams: { filter: filter || null },
          queryParamsHandling: 'merge',
        });
        this.filterChanged.emit(filter);
      });
  }

  ngOnDestroy() {
    this.$unsubscribe.next();
    this.$unsubscribe.complete();
  }

}
