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

import {StateParams} from './daemonsetdetail_state';
import {getDaemonSetDetailsResource} from './daemonsetdetail_stateconfig';

/**
 * Controller for the delete daemonset dialog.
 *
 * @final
 */
export default class DeleteDaemonSetDialogController {
  /**
   * @param {!md.$dialog} $mdDialog
   * @param {!angular.$resource} $resource
   * @param {string} namespace
   * @param {string} daemonSet
   * @ngInject
   */
  constructor($mdDialog, $resource, namespace, daemonSet) {
    /** @export {string} */
    this.daemonSet = daemonSet;

    /** @export {string} */
    this.namespace = namespace;

    /** @export {boolean} */
    this.deleteServices = false;

    /** @private {!md.$dialog} */
    this.mdDialog_ = $mdDialog;

    /** @private {!angular.$resource} */
    this.resource_ = $resource;
  }

  /**
   * Deletes the daemonset and closes the dialog.
   *
   * @export
   */
  remove() {
    let resource = getDaemonSetDetailsResource(
        new StateParams(this.namespace, this.daemonSet), this.resource_);

    /** @type {!backendApi.DeleteDaemonSetSpec} */
    let deleteDaemonSetSpec = {
      deleteServices: this.deleteServices,
    };

    resource.remove(
        deleteDaemonSetSpec, () => { this.mdDialog_.hide(); },
        () => { this.mdDialog_.cancel(); });
  }

  /**
   * Cancels and closes the dialog.
   *
   * @export
   */
  cancel() { this.mdDialog_.cancel(); }
}
