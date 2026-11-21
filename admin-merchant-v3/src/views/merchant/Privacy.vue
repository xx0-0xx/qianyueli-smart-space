<template>
  <div class="legal-page">
    <div class="legal-shell">
      <div class="legal-toolbar">
        <el-button :icon="ArrowLeft" @click="handleBack">返回</el-button>
      </div>

      <article class="legal-doc">
        <h1>隐私政策</h1>
        <p class="legal-meta">生效日期：2026年10月14日</p>

        <p>
          阡月里（以下简称"我们"）非常重视用户个人信息与隐私保护。本政策说明在您使用「阡月里」商家后台小程序（以下简称"本小程序"）时，我们如何收集、使用、存储、共享和保护您的信息，以及您所享有的权利。请您仔细阅读并同意本政策后再使用本服务。
        </p>

        <h2>一、我们收集的信息</h2>
        <p>
          1. 账号与身份信息：您登录时使用的门店邀请码，以及由微信平台生成的用户标识（OpenID），用于识别商家身份并绑定门店。
        </p>
        <p>
          2. 门店资料：您主动填写的门店名称、地址、WiFi 名称与密码、客服联系电话等信息，用于门店经营及向顾客展示。
        </p>
        <p>
          3. 经营与操作信息：订单、房态、营收数据，以及设备控制与操作日志，用于提供经营管理和设备控制服务。
        </p>
        <p>4. 设备与日志信息：设备标识、网络地址、操作时间等，用于保障服务安全与故障排查。</p>

        <h2>二、信息的使用目的与方式</h2>
        <p>
          我们仅在为您提供商家后台服务所必需的范围内使用上述信息，包括：登录鉴权与门店绑定、门店与设备管理、订单与营收统计、客户服务与安全风控。我们不会将上述信息用于与提供服务无关的用途。
        </p>

        <h2>三、信息的存储与保护</h2>
        <p>1. 存储地点：您的信息存储于中华人民共和国境内的服务器。</p>
        <p>2. 存储期限：在为您提供服务所需的期限内保存；法律法规另有规定的，从其规定。</p>
        <p>
          3. 安全措施：我们采用加密传输、访问控制等措施保护您的信息，防止未经授权的访问、泄露与篡改。
        </p>

        <h2>四、信息的共享、转让与公开披露</h2>
        <p>1. 我们不会向任何第三方出售您的个人信息。</p>
        <p>
          2. 为提供服务，我们可能向以下第三方提供必要信息：微信开放平台（登录与授权）、云服务器及消息通信服务商（数据存储与设备通信）。我们要求其按照本政策及相关法律法规处理信息。
        </p>
        <p>3. 在法律法规要求，或司法、行政机关依法要求时，我们可能依法提供相关信息。</p>

        <h2>五、您的权利</h2>
        <p>
          您有权查询、更正、删除您的个人信息，撤回授权同意，或注销账号。如需行使上述权利，可通过本政策末尾的联系方式与我们联系，我们将在核实您的身份后于合理期限内处理。
        </p>

        <h2>六、未成年人保护</h2>
        <p>本小程序面向商家用户（成年人）。若您为未成年人，请在监护人陪同下阅读本政策并使用本服务。</p>

        <h2>七、政策的更新</h2>
        <p>
          我们可能适时更新本政策，更新后将在本小程序内公布。若您在更新后继续使用本服务，即视为您接受更新后的政策。
        </p>

        <h2>八、联系我们</h2>
        <p>如您对本政策或个人信息保护有任何疑问、意见或投诉，请通过以下方式联系我们：</p>
        <p>邮箱：ck33678@outlook.com</p>
      </article>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useRouter } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'

defineOptions({ name: 'MerchantPrivacy' })

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
