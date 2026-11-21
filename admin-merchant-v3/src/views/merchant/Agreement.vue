<template>
  <div class="legal-page">
    <div class="legal-shell">
      <div class="legal-toolbar">
        <el-button :icon="ArrowLeft" @click="handleBack">返回</el-button>
      </div>

      <article class="legal-doc">
        <h1>用户服务协议</h1>
        <p class="legal-meta">生效日期：2026年10月14日</p>

        <p>
          欢迎使用「阡月里」商家后台小程序（以下简称"本小程序"）。本协议由您与阡月里（以下简称"我们"）共同缔结。请您在使用本小程序前，仔细阅读并充分理解本协议全部内容。当您勾选同意并使用本小程序时，即表示您已阅读、理解并同意接受本协议的全部条款。
        </p>

        <h2>一、协议的范围</h2>
        <p>1. 本协议是您就使用本小程序及其中各项服务所订立的协议。</p>
        <p>
          2. 我们可能根据业务需要更新本协议，更新后将在本小程序内公布。若您在更新后继续使用，视为您接受更新后的内容。
        </p>

        <h2>二、账号与登录</h2>
        <p>
          1. 本小程序通过门店邀请码及微信授权识别商家身份。您应妥善保管邀请码，因保管不善造成的损失由您自行承担。
        </p>
        <p>2. 您承诺以真实、合法的方式使用本服务，并对通过您账号进行的一切操作负责。</p>

        <h2>三、服务内容</h2>
        <p>
          本小程序为您提供门店管理、房型与房间管理、设备绑定与控制、订单与营收查看、充值确认、优惠券管理等服务。具体功能以本小程序实际提供为准。
        </p>

        <h2>四、您的权利与义务</h2>
        <p>1. 您有权按照本协议的约定使用本服务。</p>
        <p>
          2. 您不得利用本服务从事任何违反法律法规、侵犯他人合法权益的行为，不得干扰或破坏本服务的正常运行。
        </p>
        <p>3. 您应保证所填写、上传的门店信息真实、准确、合法，因信息不实产生的责任由您承担。</p>

        <h2>五、知识产权</h2>
        <p>
          本小程序及其相关的软件、技术、界面、文字、图形等知识产权归我们或相关权利人所有。未经许可，您不得复制、传播、改编或用于其他商业用途。
        </p>

        <h2>六、服务的中断、中止与终止</h2>
        <p>1. 因系统维护、升级、故障或不可抗力等原因，本服务可能中断或暂停，我们将尽力及时恢复。</p>
        <p>2. 若您违反本协议或相关法律法规，我们有权中止或终止向您提供服务。</p>

        <h2>七、免责声明</h2>
        <p>
          在法律允许的范围内，因您自身原因、第三方原因或不可抗力导致的服务中断、数据丢失等，我们不承担相应责任，但我们将尽合理努力协助处理。
        </p>

        <h2>八、法律适用与争议解决</h2>
        <p>
          本协议的订立、效力、解释及争议解决均适用中华人民共和国法律。因本协议产生的争议，双方应友好协商解决；协商不成的，任何一方可向有管辖权的人民法院提起诉讼。
        </p>

        <h2>九、联系我们</h2>
        <p>如您对本协议有任何疑问，可通过邮箱 ck33678@outlook.com 与我们联系。</p>
      </article>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useRouter } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'

defineOptions({ name: 'MerchantAgreement' })

const router = useRouter()

const handleBack = () => {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.replace('/store-config')
  }
}
</script>

<style lang="scss" scoped>
.legal-page {
  min-height: 100vh;
  padding: 32px 20px 60px;
  background: var(--el-bg-color-page);
}

.legal-shell {
  max-width: 900px;
  margin: 0 auto;
}

.legal-toolbar {
  margin-bottom: 14px;
}

.legal-doc {
  padding: 48px 56px;
  color: var(--el-text-color-primary);
  font-size: 15px;
  line-height: 1.9;
  word-break: break-word;
  background: var(--el-bg-color);
  border-radius: 14px;
  box-shadow: 0 8px 30px rgb(15 23 42 / 6%);
}

.legal-doc h1 {
  margin: 0 0 10px;
  font-size: 28px;
  font-weight: 700;
  line-height: 1.4;
  color: var(--el-text-color-primary);
  text-align: center;
  letter-spacing: 1px;
}

.legal-meta {
  margin: 0 0 30px;
  font-size: 13px;
  color: var(--el-text-color-placeholder);
  text-align: center;
}

.legal-doc p {
  margin: 0 0 12px;
  text-align: justify;
}

.legal-doc h2 {
  margin: 28px 0 12px;
  padding-left: 10px;
  font-size: 17px;
  font-weight: 600;
  line-height: 1.5;
  color: var(--el-text-color-primary);
  border-left: 3px solid #1a56db;
}

@media (max-width: 640px) {
  .legal-page {
    padding: 16px 12px 40px;
  }

  .legal-doc {
    padding: 26px 20px;
    font-size: 14px;
  }

  .legal-doc h1 {
    font-size: 22px;
  }

  .legal-doc h2 {
    font-size: 16px;
  }
}
</style>
