// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import Breadcrumb from './breadcrumb';

/** Breadcrumbs config string used on state config. **/
export const breadcrumbsConfig = 'kdBreadcrumbs';

/**
 * @final
 */
export default class BreadcrumbsController {
  /**
   * Constructs breadcrumbs controller.
   *
   * @param {!ui.router.$state} $state
   * @param {!angular.$interpolate} $interpolate
   * @ngInject
   */
  constructor($state, $interpolate) {
    /** @private {!ui.router.$state} */
    this.state_ = $state;

    /** @private {!angular.$interpolate} */
    this.interpolate_ = $interpolate;

    /** @export {number} - Initialized from template */
    this.limit;

    /** @export {!Array<!Breadcrumb>} - Initialized in $onInit method. Used in template */
    this.breadcrumbs;
  }

  /**
   * @export
   */
  $onInit() { this.breadcrumbs = this.initBreadcrumbs_(); }

  /**
   * Initializes breadcrumbs array by traversing states parents until none is found.
   *
   * @return {!Array<!Breadcrumb>}
   * @private
   */
  initBreadcrumbs_() {
    /** @type {!ui.router.$state} */
    let state = this.state_['$current'];
    /** @type {!Array<!Breadcrumb>} */
    let breadcrumbs = [];

    while (state && state['name'] && this.canAddBreadcrumb_(breadcrumbs)) {
      /** @type {!Breadcrumb} */
      let breadcrumb = this.getBreadcrumb_(state);

      if (breadcrumb.label) {
        breadcrumbs.push(breadcrumb);
      }

      state = this.getParentState_(state);
    }

    return breadcrumbs.reverse();
  }

  /**
   * Returns true if limit is undefined or limit is defined and breadcrumbs count is smaller or
   * equal to the limit.
   *
   * @param {!Array<!Breadcrumb>} breadcrumbs
   * @return {boolean}
   * @private
   */
  canAddBreadcrumb_(breadcrumbs) {
    return this.limit === undefined || this.limit > breadcrumbs.length;
  }

  /**
   * Returns breadcrumb config object if it is defined on state, undefined otherwise.
   *
   * @param {!ui.router.$state} state
   * @return {Object}
   * @private
   */
  getBreadcrumbConfig_(state) {
    let conf = state['data'];

    if (conf) {
      conf = conf[breadcrumbsConfig];
    }

    return conf;
  }

  /**
   * Returns parent state of the given state based on defined state parent name or if it is not
   * defined then based on direct parent state.
   *
   * @param {!ui.router.$state} state
   * @return {!ui.router.$state}
   * @private
   */
  getParentState_(state) {
    let conf = this.getBreadcrumbConfig_(state);
    let result = state['parent'];

    if (conf && conf.parent) {
      result = this.state_.get(conf.parent);
    }

    return result;
  }

  /**
   * Creates breadcrumb object based on state object.
   *
   * @param {!ui.router.$state} state
   * @return {!Breadcrumb}
   * @private
   */
  getBreadcrumb_(state) {
    let breadcrumb = new Breadcrumb();

    breadcrumb.label = this.getDisplayName_(state);
    breadcrumb.stateLink = this.state_.href(state['name']);

    return breadcrumb;
  }

  /**
   * Returns display name for given state.
   *
   * If label for the state is defined it is interpolated against given state default view scope.
   * If interpolation fails then given label string is returned.
   * If label string is empty then state name is returned.
   *
   * @param {!ui.router.$state} state
   * @returns {string}
   * @private
   */
  getDisplayName_(state) {
    let conf = this.getBreadcrumbConfig_(state);
    let areLocalsDefined = !!state['locals'];
    let interpolationContext = state;

    // When conf is undefined and label is undefined or empty then fallback to state name
    if (!conf || !conf.label) {
      return state['name'];
    }

    if (areLocalsDefined) {
      // Set context to default view scope
      interpolationContext = state['locals']['@'];
    }

    return this.interpolate_(conf.label)(interpolationContext).toString();
  }
}

/**
 * Returns breadcrumbs component. Should be used only withing actionbar component.
 *
 * In order to define custom label for the state add: `'kdBreadcrumbs':{'label':'myLabel'}`
 * to the state config. This label will be used instead of default state name when displaying
 * breadcrumbs.
 *
 * In order to define custom parent for the state add: `'kdBreadcrumbs`:{'parent':'myParentState'}`
 * to the state config. Parent state will be looked up by this name if it's defined.
 *
 * Additionally labels can be interpolated. This applies only to the last state in the
 * breadcrumb chain, i.e. for given state chain `StateA > StateB > StateC`, only StateC can use
 * following convention: `'kdBreadcrumbs`:{'label':'{{$stateParams.paramX}}'}`
 *
 * Example state config:
 * $stateProvider.state(stateName, {
 *   url: '...',
 *   data: {
 *      'kdBreadcrumbs': {
 *        'label': '{{$stateParams.paramX}}',
 *        'parent': 'parentState',
 *      },
 *   },
 *   views: {
 *     '': {
 *       controller: SomeCtrlA,
 *       controllerAs: 'ctrl',
 *       templateUrl: '...',
 *     },
 *     'actionbar': {
 *       controller: SomeCtrlB,
 *       controllerAs: 'ctrl',
 *       templateUrl: '...',
 *     },
 *   },
 * });
 *
 * @return {!angular.Component}
 */
export const breadcrumbsComponent = {
  binding: {
    'limit': '<',
  },
  controller: BreadcrumbsController,
  controllerAs: 'ctrl',
  templateUrl: 'common/components/breadcrumbs/breadcrumbs.html',
};
