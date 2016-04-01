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

import {StateParams} from 'daemonsetdetail/daemonsetdetail_state';
import {stateName} from 'daemonsetdetail/daemonsetdetail_state';

/**
 * Controller for the daemonset card menu
 *
 * @final
 */
export default class DaemonSetCardMenuController {
  /**
   * @param {!ui.router.$state} $state
   * @param
   * {!./../daemonsetdetail/daemonset_service.DaemonSetService}
   * kdDaemonSetService
   * @ngInject
   */
  constructor($state, kdDaemonSetService) {
    /**
     * Initialized from the scope.
     * @export {!backendApi.DaemonSet}
     */
    this.daemonSet;

    /** @private {!ui.router.$state} */
    this.state_ = $state;

    /** @private
     * {!./../daemonsetdetail/daemonset_service.DaemonSetService}
     */
    this.kdDaemonSetService_ = kdDaemonSetService;
  }

  /**
   * @param {!function(!MouseEvent)} $mdOpenMenu
   * @param {!MouseEvent} $event
   * @export
   */
  openMenu($mdOpenMenu, $event) { $mdOpenMenu($event); }

  /**
   * @export
   */
  viewDetails() {
    this.state_.go(
        stateName,
        new StateParams(this.daemonSet.namespace, this.daemonSet.name));
  }

  /**
   * @export
   */
  showDeleteDialog() {
    this.kdDaemonSetService_
        .showDeleteDialog(this.daemonSet.namespace, this.daemonSet.name)
        .then(() => this.state_.reload());
  }

}
